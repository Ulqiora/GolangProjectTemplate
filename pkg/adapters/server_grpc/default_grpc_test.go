package server_grpc

import (
	"testing"

	"google.golang.org/grpc"
)

func TestGrpcServerRegisterAndBaseServer(t *testing.T) {
	t.Parallel()

	srv := NewGrpcServer(ServerConfig{})
	if srv.BaseServer() == nil {
		t.Fatal("expected base grpc server")
	}

	called := 0
	srv.Register(nil, func(base *grpc.Server) {
		if base == nil {
			t.Fatal("expected grpc server")
		}
		called++
	})

	for _, registrar := range srv.registrars {
		registrar(srv.BaseServer())
	}

	if called != 1 {
		t.Fatalf("expected registrar to be called once, got %d", called)
	}
}
