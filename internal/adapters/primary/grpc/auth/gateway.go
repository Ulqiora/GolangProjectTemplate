package auth

import (
	"context"
	"net/http"

	authv1 "GolangTemplateProject/internal/adapters/primary/generated/auth/v1"
	"GolangTemplateProject/internal/adapters/primary/web"
	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterGateway(proxy *servergrpc.GrpcProxy) {
	if proxy == nil {
		return
	}

	proxy.SetMux(runtime.NewServeMux())
	proxy.AddService(authv1.RegisterAuthServiceHandlerFromEndpoint)
	proxy.AddConnService(func(_ context.Context, mux *runtime.ServeMux, _ *grpc.ClientConn) error {
		return mux.HandlePath(http.MethodGet, "/healthz", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	})
	proxy.AddConnService(func(_ context.Context, mux *runtime.ServeMux, _ *grpc.ClientConn) error {
		handler := web.Handler()
		for _, path := range []string{"/", "/styles.css", "/app.js", "/anime-hero.png", "/yandex/callback"} {
			if err := mux.HandlePath(http.MethodGet, path, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
				handler.ServeHTTP(w, r)
			}); err != nil {
				return err
			}
		}
		return nil
	})
	proxy.AddConnService(func(_ context.Context, mux *runtime.ServeMux, _ *grpc.ClientConn) error {
		handler := web.AnimeCatalogHandler()
		return mux.HandlePath(http.MethodGet, "/v1/anime/catalog", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			handler.ServeHTTP(w, r)
		})
	})
}
