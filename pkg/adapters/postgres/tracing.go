package postgres

import (
	"context"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type pgxTracer struct {
	tracer trace.Tracer
	attrs  []attribute.KeyValue
}

type querySpanKey struct{}
type batchSpanKey struct{}
type copySpanKey struct{}
type prepareSpanKey struct{}
type connectSpanKey struct{}
type acquireSpanKey struct{}

func newPGXTracer(provider trace.TracerProvider, role string, cfg EndpointConfig) *pgxTracer {
	if provider == nil {
		provider = otel.GetTracerProvider()
	}
	return &pgxTracer{
		tracer: provider.Tracer("GolangTemplateProject/pkg/adapters/postgres"),
		attrs: []attribute.KeyValue{
			attribute.String("db.system", "postgresql"),
			attribute.String("db.name", cfg.Database),
			attribute.String("db.user", cfg.User),
			attribute.String("db.connection.role", role),
			attribute.String("net.peer.name", cfg.Host),
			attribute.String("net.peer.port", cfg.Port),
		},
	}
}

func (t *pgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "postgres.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.queryAttrs(data.SQL)...),
	)
	return context.WithValue(ctx, querySpanKey{}, span)
}

func (t *pgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	endSpan(ctx, querySpanKey{}, data.Err)
}

func (t *pgxTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	attrs := append([]attribute.KeyValue{}, t.attrs...)
	if data.Batch != nil {
		attrs = append(attrs, attribute.Int("db.batch.size", data.Batch.Len()))
	}
	ctx, span := t.tracer.Start(ctx, "postgres.batch",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return context.WithValue(ctx, batchSpanKey{}, span)
}

func (t *pgxTracer) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchQueryData) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		if stored, ok := ctx.Value(batchSpanKey{}).(trace.Span); ok {
			span = stored
		}
	}
	if span == nil || !span.SpanContext().IsValid() {
		return
	}
	attrs := t.queryAttrs(data.SQL)
	if data.Err != nil {
		attrs = append(attrs, attribute.String("error", data.Err.Error()))
	}
	span.AddEvent("postgres.batch.query", trace.WithAttributes(attrs...))
}

func (t *pgxTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	endSpan(ctx, batchSpanKey{}, data.Err)
}

func (t *pgxTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	attrs := append([]attribute.KeyValue{}, t.attrs...)
	attrs = append(attrs,
		attribute.String("db.operation", "COPY"),
		attribute.String("db.sql.table", strings.Join(data.TableName, ".")),
	)
	ctx, span := t.tracer.Start(ctx, "postgres.copy_from",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return context.WithValue(ctx, copySpanKey{}, span)
}

func (t *pgxTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	endSpan(ctx, copySpanKey{}, data.Err)
}

func (t *pgxTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	attrs := t.queryAttrs(data.SQL)
	attrs = append(attrs, attribute.String("db.statement.name", data.Name))
	ctx, span := t.tracer.Start(ctx, "postgres.prepare",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return context.WithValue(ctx, prepareSpanKey{}, span)
}

func (t *pgxTracer) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	endSpan(ctx, prepareSpanKey{}, data.Err)
}

func (t *pgxTracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "postgres.connect",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	)
	return context.WithValue(ctx, connectSpanKey{}, span)
}

func (t *pgxTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	endSpan(ctx, connectSpanKey{}, data.Err)
}

func (t *pgxTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "postgres.acquire",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.attrs...),
	)
	return context.WithValue(ctx, acquireSpanKey{}, span)
}

func (t *pgxTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	endSpan(ctx, acquireSpanKey{}, data.Err)
}

func (t *pgxTracer) TraceRelease(_ *pgxpool.Pool, _ pgxpool.TraceReleaseData) {}

func (t *pgxTracer) queryAttrs(sql string) []attribute.KeyValue {
	attrs := append([]attribute.KeyValue{}, t.attrs...)
	attrs = append(attrs,
		attribute.String("db.operation", sqlOperation(sql)),
		attribute.String("db.statement", truncateStatement(sql)),
	)
	return attrs
}

func endSpan(ctx context.Context, key any, err error) {
	span, ok := ctx.Value(key).(trace.Span)
	if !ok || span == nil {
		return
	}
	defer span.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetStatus(codes.Ok, "ok")
}

func sqlOperation(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "UNKNOWN"
	}
	fields := strings.FieldsFunc(sql, func(r rune) bool {
		return unicode.IsSpace(r) || r == '('
	})
	if len(fields) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(fields[0])
}

func truncateStatement(sql string) string {
	const limit = 2048
	sql = strings.TrimSpace(sql)
	if len(sql) <= limit {
		return sql
	}
	return sql[:limit] + "..."
}
