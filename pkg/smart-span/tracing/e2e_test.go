package tracing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testJaegerImage     = "jaegertracing/all-in-one:1.57"
	testJaegerOTLPPort  = "4318/tcp"
	testJaegerQueryPort = "16686/tcp"
)

type jaegerTraceSearchResponse struct {
	Data []jaegerTrace `json:"data"`
}

type jaegerTrace struct {
	TraceID string       `json:"traceID"`
	Spans   []jaegerSpan `json:"spans"`
}

type jaegerSpan struct {
	TraceID       string `json:"traceID"`
	OperationName string `json:"operationName"`
}

func TestE2ESmartSpanMiniProjectPublishesTraceToJaeger(t *testing.T) {
	if testing.Short() {
		t.Skip("tracing e2e test is skipped in short mode")
	}

	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	host, otlpPort, queryPort := startTestJaeger(t, ctx)
	serviceName := "smart-span-e2e-" + uuid.NewString()

	tracerProvider, shutdown, err := InitTracing(ctx, &TracerConfig{
		Endpoint:    netJoinHostPort(host, otlpPort),
		ServiceName: serviceName,
		Timeout:     5,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, shutdown(context.Background()))
	})

	traceID := runMiniProjectFlow(context.Background())
	require.NoError(t, tracerProvider.ForceFlush(ctx))

	foundTrace := waitForTraceInJaeger(t, host, queryPort, serviceName)
	require.Equal(t, traceID, foundTrace.TraceID)
	require.ElementsMatch(t, []string{"mini-project-root", "mini-project-child"}, traceOperations(foundTrace))

	t.Logf("trace is available in Jaeger UI: http://%s:%s/search?service=%s", host, queryPort, serviceName)
	t.Logf("trace id: %s", traceID)
}

func runMiniProjectFlow(ctx context.Context) string {
	ctx, rootSpan := SetName("mini-project-root").Start(ctx)
	defer rootSpan.End()

	rootSpan.AddEvent("mini-project-started")

	_, childSpan := GetDefaultTracer().Start(ctx, "mini-project-child")
	childSpan.RecordError(errors.New("mini-project-error"))
	childSpan.End()

	return rootSpan.SpanContext().TraceID().String()
}

func startTestJaeger(t *testing.T, ctx context.Context) (string, string, string) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skipping e2e tracing test: Docker/testcontainers panic: %v", r)
		}
	}()

	req := tc.GenericContainerRequest{
		Started: true,
		ContainerRequest: tc.ContainerRequest{
			Image: testJaegerImage,
			Env: map[string]string{
				"COLLECTOR_OTLP_ENABLED": "true",
			},
			ExposedPorts: []string{testJaegerOTLPPort, testJaegerQueryPort},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort(nat.Port(testJaegerOTLPPort)),
				wait.ForHTTP("/api/services").WithPort(nat.Port(testJaegerQueryPort)),
			).WithStartupTimeout(2 * time.Minute),
		},
	}

	container, err := tc.GenericContainer(ctx, req)
	if err != nil {
		t.Skipf("skipping e2e tracing test: Docker/testcontainers unavailable: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	otlpPort, err := container.MappedPort(ctx, nat.Port(testJaegerOTLPPort))
	require.NoError(t, err)

	queryPort, err := container.MappedPort(ctx, nat.Port(testJaegerQueryPort))
	require.NoError(t, err)

	return host, otlpPort.Port(), queryPort.Port()
}

func waitForTraceInJaeger(t *testing.T, host string, queryPort string, serviceName string) jaegerTrace {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	var result jaegerTrace

	require.Eventually(t, func() bool {
		trace, ok := fetchTraceFromJaeger(t, client, host, queryPort, serviceName)
		if ok {
			result = trace
		}
		return ok
	}, 30*time.Second, 1*time.Second)

	return result
}

func fetchTraceFromJaeger(t *testing.T, client *http.Client, host string, queryPort string, serviceName string) (jaegerTrace, bool) {
	t.Helper()

	endpoint := fmt.Sprintf(
		"http://%s:%s/api/traces?service=%s&operation=%s&limit=20",
		host,
		queryPort,
		url.QueryEscape(serviceName),
		url.QueryEscape("mini-project-root"),
	)

	resp, err := client.Get(endpoint)
	if err != nil {
		return jaegerTrace{}, false
	}
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return jaegerTrace{}, false
	}

	var payload jaegerTraceSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return jaegerTrace{}, false
	}

	if len(payload.Data) == 0 {
		return jaegerTrace{}, false
	}

	return payload.Data[0], true
}

func traceOperations(trace jaegerTrace) []string {
	operations := make([]string, 0, len(trace.Spans))
	for _, span := range trace.Spans {
		operations = append(operations, span.OperationName)
	}
	return operations
}

func netJoinHostPort(host string, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}
