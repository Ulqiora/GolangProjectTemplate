package server_grpc

import (
	"context"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceRegistry func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

type GrpcProxy struct {
	mux             *runtime.ServeMux
	grpcProxyConfig ServerConfig
	options         []grpc.DialOption
	services        []ServiceRegistry
}

func (r *GrpcProxy) AddService(service ServiceRegistry) *GrpcProxy {
	r.services = append(r.services, service)
	return r
}

func NewProxy(grpcProxyConfig ServerConfig) *GrpcProxy {
	// Создаем мультиплексор для gRPC-Gateway
	var grpcProxy GrpcProxy
	grpcProxy.mux = runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   runtime.JSONPb{}.MarshalOptions,
			UnmarshalOptions: runtime.JSONPb{}.UnmarshalOptions,
		}),
	)
	// Регистрируем обработчики
	grpcProxy.options = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024)),
	}
	grpcProxy.grpcProxyConfig = grpcProxyConfig
	//err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	//if err != nil {
	//	return err
	//}
	//
	//// Настраиваем HTTP сервер
	//httpServer := &http.Server{
	//	Addr:         ":8080",
	//	Handler:      mux,
	//	ReadTimeout:  15 * time.Second,
	//	WriteTimeout: 15 * time.Second,
	//}
	return nil
}

func (r *GrpcProxy) Start(ctx context.Context, group *sync.WaitGroup) error {
	for _, service := range r.services {
		err := service(ctx, r.mux, r.grpcProxyConfig.String(), r.options)
		if err != nil {
			return err
		}
	}
	httpMux := http.NewServeMux()
	httpMux.Handle("/", r.mux)

	// Health check
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Метрики (опционально)
	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
	})
	server := &http.Server{
		Addr:    r.grpcProxyConfig.String(),
		Handler: httpMux,
	}
	return server.ListenAndServe()
}
