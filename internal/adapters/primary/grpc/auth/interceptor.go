package auth

import (
	"context"
	"time"

	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	"github.com/prometheus/client_golang/prometheus"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type validator interface {
	Validate() error
}

type validatorAll interface {
	ValidateAll() error
}

type serverMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewUnaryServerInterceptor(log logger.Logger, registerer prometheus.Registerer) grpc.UnaryServerInterceptor {
	baseLogger := log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}

	metrics := newServerMetrics(registerer)
	componentLogger := baseLogger.WithN("grpc_auth")

	return func(
		ctx context.Context,
		request interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startedAt := time.Now()
		ctx = servergrpc.ExtractIncomingTraceContext(ctx)
		ctx, span := smarttracing.GetDefaultTracer().Start(ctx, info.FullMethod, trace.WithAttributes(
			otelattribute.String("rpc.system", "grpc"),
			otelattribute.String("rpc.method", info.FullMethod),
		))
		defer span.End()

		if err := validateRequest(request); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			componentLogger.Warn("gRPC request validation failed",
				attribute.String("method", info.FullMethod),
				attribute.String("code", codes.InvalidArgument.String()),
				attribute.String("error", err.Error()),
			)
			metrics.observe(info.FullMethod, codes.InvalidArgument, startedAt)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		response, err := handler(ctx, request)
		code := status.Code(err)

		fields := []attribute.Field{
			attribute.String("method", info.FullMethod),
			attribute.String("code", code.String()),
			attribute.Float64("duration_seconds", time.Since(startedAt).Seconds()),
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			fields = append(fields, attribute.String("error", err.Error()))
			if code == codes.Internal {
				componentLogger.Error("gRPC request failed", fields...)
			} else {
				componentLogger.Warn("gRPC request failed", fields...)
			}
		} else {
			span.SetStatus(otelcodes.Ok, "ok")
			componentLogger.Info("gRPC request handled", fields...)
		}

		metrics.observe(info.FullMethod, code, startedAt)
		return response, err
	}
}

func validateRequest(request interface{}) error {
	switch message := request.(type) {
	case validatorAll:
		return message.ValidateAll()
	case validator:
		return message.Validate()
	default:
		return nil
	}
}

func newServerMetrics(registerer prometheus.Registerer) *serverMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "auth_grpc",
			Name:      "requests_total",
			Help:      "Total number of gRPC auth requests by method and code.",
		},
		[]string{"method", "code"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "auth_grpc",
			Name:      "request_duration_seconds",
			Help:      "Duration of gRPC auth requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "code"},
	)

	if err := registerer.Register(requests); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				requests = existing
			}
		}
	}
	if err := registerer.Register(duration); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				duration = existing
			}
		}
	}

	return &serverMetrics{requests: requests, duration: duration}
}

func (m *serverMetrics) observe(method string, code codes.Code, startedAt time.Time) {
	labelCode := code.String()
	m.requests.WithLabelValues(method, labelCode).Inc()
	m.duration.WithLabelValues(method, labelCode).Observe(time.Since(startedAt).Seconds())
}
