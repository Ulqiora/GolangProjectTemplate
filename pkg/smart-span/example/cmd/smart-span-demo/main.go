package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"GolangTemplateProject/pkg/smart-span/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type appConfig struct {
	serviceName      string
	httpPort         string
	jaegerUIURL      string
	otlpEndpoint     string
	autoFlowInterval time.Duration
}

type flowResponse struct {
	ServiceName   string `json:"service_name"`
	TraceID       string `json:"trace_id"`
	Flow          string `json:"flow"`
	Success       bool   `json:"success"`
	FailRequested bool   `json:"fail_requested"`
	JaegerTrace   string `json:"jaeger_trace_url"`
	JaegerSearch  string `json:"jaeger_search_url"`
	Error         string `json:"error,omitempty"`
}

type rootResponse struct {
	ServiceName      string   `json:"service_name"`
	JaegerUI         string   `json:"jaeger_ui"`
	AvailableRoutes  []string `json:"available_routes"`
	HowToSeeTheTrace []string `json:"how_to_see_the_trace"`
}

func main() {
	cfg := loadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracerProvider, shutdown, err := tracing.InitTracing(ctx, &tracing.TracerConfig{
		Endpoint:    cfg.otlpEndpoint,
		ServiceName: cfg.serviceName,
		Timeout:     5,
	})
	if err != nil {
		log.Fatalf("init tracing: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown tracing: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler(cfg))
	mux.HandleFunc("/demo/checkout", checkoutHandler(cfg, tracerProvider))

	server := &http.Server{
		Addr:              ":" + cfg.httpPort,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go runBackgroundFlows(ctx, cfg, tracerProvider)

	go func() {
		log.Printf("smart-span demo is listening on http://localhost:%s", cfg.httpPort)
		log.Printf("jaeger ui is available on %s", cfg.jaegerUIURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown: %v", err)
	}
}

func loadConfig() appConfig {
	return appConfig{
		serviceName:      envOrDefault("SMART_SPAN_SERVICE_NAME", "smart-span-demo"),
		httpPort:         envOrDefault("SMART_SPAN_HTTP_PORT", "18080"),
		jaegerUIURL:      envOrDefault("SMART_SPAN_JAEGER_UI", "http://localhost:16686"),
		otlpEndpoint:     envOrDefault("SMART_SPAN_OTLP_ENDPOINT", "jaeger:4318"),
		autoFlowInterval: parseDurationOrDefault("SMART_SPAN_AUTO_FLOW_INTERVAL", 5*time.Second),
	}
}

func rootHandler(cfg appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, rootResponse{
			ServiceName: cfg.serviceName,
			JaegerUI:    cfg.jaegerUIURL,
			AvailableRoutes: []string{
				"GET /demo/checkout",
				"GET /demo/checkout?fail=true",
			},
			HowToSeeTheTrace: []string{
				"open Jaeger UI",
				"search for service " + cfg.serviceName,
				"open the latest trace",
				"or call /demo/checkout and use jaeger_trace_url from response",
			},
		})
	}
}

func checkoutHandler(cfg appConfig, tracerProvider *sdktrace.TracerProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		failRequested := strings.EqualFold(r.URL.Query().Get("fail"), "true")
		traceID, err := runCheckoutFlow(r.Context(), cfg.serviceName, failRequested)
		flushTracing(r.Context(), tracerProvider)
		response := flowResponse{
			ServiceName:   cfg.serviceName,
			TraceID:       traceID,
			Flow:          "checkout",
			Success:       err == nil,
			FailRequested: failRequested,
			JaegerTrace:   cfg.jaegerUIURL + "/trace/" + traceID,
			JaegerSearch:  cfg.jaegerUIURL + "/search",
		}
		if err != nil {
			response.Error = err.Error()
			writeJSON(w, http.StatusInternalServerError, response)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func runBackgroundFlows(ctx context.Context, cfg appConfig, tracerProvider *sdktrace.TracerProvider) {
	if cfg.autoFlowInterval <= 0 {
		return
	}

	ticker := time.NewTicker(cfg.autoFlowInterval)
	defer ticker.Stop()

	failNext := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			traceID, err := runCheckoutFlow(context.Background(), cfg.serviceName, failNext)
			flushTracing(context.Background(), tracerProvider)
			if err != nil {
				log.Printf("background checkout failed trace_id=%s err=%v", traceID, err)
			} else {
				log.Printf("background checkout completed trace_id=%s", traceID)
			}
			failNext = !failNext
		}
	}
}

func flushTracing(ctx context.Context, tracerProvider *sdktrace.TracerProvider) {
	if tracerProvider == nil {
		return
	}

	flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := tracerProvider.ForceFlush(flushCtx); err != nil {
		log.Printf("force flush tracing: %v", err)
	}
}

func runCheckoutFlow(ctx context.Context, serviceName string, failPayment bool) (string, error) {
	ctx, rootSpan := tracing.SetName("checkout.flow").Start(ctx)
	defer rootSpan.End()

	rootSpan.SetAttributes(
		attribute.String("demo.service", serviceName),
		attribute.Bool("demo.fail_payment", failPayment),
	)
	rootSpan.AddEvent("checkout.started")

	traceID := rootSpan.SpanContext().TraceID().String()

	if err := validateCustomer(ctx); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return traceID, err
	}
	if err := reserveInventory(ctx); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return traceID, err
	}
	if err := chargePayment(ctx, failPayment); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		rootSpan.AddEvent("checkout.failed")
		return traceID, err
	}
	if err := publishOrderEvent(ctx); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return traceID, err
	}

	rootSpan.SetStatus(codes.Ok, "checkout completed")
	rootSpan.AddEvent("checkout.completed")
	return traceID, nil
}

func validateCustomer(ctx context.Context) error {
	_, span := tracing.GetDefaultTracer().Start(ctx, "checkout.validate_customer")
	defer span.End()

	time.Sleep(120 * time.Millisecond)
	span.AddEvent("customer.validated")
	span.SetAttributes(attribute.String("demo.customer_tier", "gold"))
	span.SetStatus(codes.Ok, "customer validated")
	return nil
}

func reserveInventory(ctx context.Context) error {
	_, span := tracing.GetDefaultTracer().Start(ctx, "checkout.reserve_inventory")
	defer span.End()

	time.Sleep(180 * time.Millisecond)
	span.AddEvent("inventory.reserved")
	span.SetAttributes(attribute.Int("demo.inventory_items", 3))
	span.SetStatus(codes.Ok, "inventory reserved")
	return nil
}

func chargePayment(ctx context.Context, fail bool) error {
	_, span := tracing.GetDefaultTracer().Start(ctx, "checkout.charge_payment")
	defer span.End()

	time.Sleep(220 * time.Millisecond)
	span.SetAttributes(attribute.String("demo.payment_provider", "demo-pay"))
	if fail {
		err := errors.New("payment gateway timeout")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.AddEvent("payment.failed")
		return err
	}

	span.AddEvent("payment.captured")
	span.SetStatus(codes.Ok, "payment captured")
	return nil
}

func publishOrderEvent(ctx context.Context) error {
	_, span := tracing.GetDefaultTracer().Start(ctx, "checkout.publish_order_event")
	defer span.End()

	time.Sleep(80 * time.Millisecond)
	span.AddEvent("order.event.published")
	span.SetAttributes(attribute.String("demo.topic", "orders.created"))
	span.SetStatus(codes.Ok, "event published")
	return nil
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %s", r.Method, r.URL.String(), time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseDurationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		return duration
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}

	log.Printf("invalid duration for %s=%q, using default %s", key, raw, fallback)
	return fallback
}
