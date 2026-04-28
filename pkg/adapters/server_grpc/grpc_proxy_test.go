package server_grpc

import (
	"context"
	"strings"
	"testing"
)

func TestGrpcProxyDisabledClosesNotify(t *testing.T) {
	t.Parallel()

	proxy := NewProxy(ProxyConfig{Enabled: false})
	if err := proxy.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	<-proxy.Notify()
}

func TestGrpcProxyRequiresEndpointAndServices(t *testing.T) {
	t.Parallel()

	proxy := NewProxy(ProxyConfig{Enabled: true, Host: "127.0.0.1", Port: 8080})
	if err := proxy.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("expected endpoint error, got %v", err)
	}

	proxy = NewProxy(ProxyConfig{Enabled: true, Host: "127.0.0.1", Port: 8080}).SetGRPCEndpoint("127.0.0.1:50051")
	if err := proxy.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "services") {
		t.Fatalf("expected services error, got %v", err)
	}
}

func TestNewAppConfiguresProxyEndpoint(t *testing.T) {
	t.Parallel()

	app, err := NewApp(AppConfig{
		GRPC:  ServerConfig{Host: "0.0.0.0", Port: 50051},
		Proxy: ProxyConfig{Enabled: true, Host: "127.0.0.1", Port: 8080},
	})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}
	if app.GRPC() == nil || app.Proxy() == nil {
		t.Fatal("expected grpc and proxy")
	}
	if app.Proxy().endpoint != "127.0.0.1:50051" {
		t.Fatalf("unexpected endpoint %q", app.Proxy().endpoint)
	}
}
