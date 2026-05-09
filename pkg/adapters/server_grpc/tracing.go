package server_grpc

import (
	"context"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TraceUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		otel.GetTextMapPropagator().Inject(ctx, metadataCarrier{md: md})
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func ExtractIncomingTraceContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier{md: md})
}

type metadataCarrier struct {
	md metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c metadataCarrier) Set(key string, value string) {
	c.md.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for key := range c.md {
		keys = append(keys, key)
	}
	return keys
}
