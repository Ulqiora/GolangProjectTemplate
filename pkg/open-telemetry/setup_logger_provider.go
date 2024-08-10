package open_telemetry

import (
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func NewLoggerProvider() (*log.LoggerProvider, error) {
	logExporter, err := stdoutlog.New()
	//logExporter, err := otlploghttp.New(context.Background())
	if err != nil {
		return nil, err
	}
	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
