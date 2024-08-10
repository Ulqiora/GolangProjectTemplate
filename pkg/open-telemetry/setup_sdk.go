package open_telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
)

var Tracer = otel.Tracer("otel")

func SetupOpenTelemetrySDK(ctx context.Context) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}
	var err error
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}
	// Set up propagator.
	//prop := NewPropagator()
	//otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider()
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	Tracer = otel.Tracer("service")

	// Set up meter provider.
	//meterProvider, err := NewMeterProvider()
	//if err != nil {
	//	handleErr(err)
	//	return shutdown, err
	//}
	//shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	//otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	//loggerProvider, err := NewLoggerProvider()
	//if err != nil {
	//	handleErr(err)
	//	return shutdown, err
	//}
	//shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	//global.SetLoggerProvider(loggerProvider)
	return shutdown, nil
}
