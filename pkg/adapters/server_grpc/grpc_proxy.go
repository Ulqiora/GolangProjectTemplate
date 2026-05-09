package server_grpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceRegistry matches grpc-gateway generated registrars like:
// RegisterXHandlerFromEndpoint(ctx, mux, endpoint, opts)
type ServiceRegistry func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// ConnRegistry is an alternative registration form using an existing grpc.ClientConn.
type ConnRegistry func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error

type GrpcProxy struct {
	cfg      ProxyConfig
	endpoint string

	mux  *runtime.ServeMux
	http *http.Server

	dialOptions []grpc.DialOption
	services    []ServiceRegistry
	connRegs    []ConnRegistry

	onceStart sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
	closeOnce sync.Once
	errCh     chan error
}

func NewProxy(cfg ProxyConfig) *GrpcProxy {
	return &GrpcProxy{
		cfg:         cfg,
		mux:         runtime.NewServeMux(),
		dialOptions: defaultProxyDialOptions(),
		errCh:       make(chan error, 1),
	}
}

func (p *GrpcProxy) SetMux(mux *runtime.ServeMux) *GrpcProxy {
	if mux != nil {
		p.mux = mux
	}
	return p
}

func (p *GrpcProxy) SetHTTPServer(srv *http.Server) *GrpcProxy {
	p.http = srv
	return p
}

// SetGRPCEndpoint sets address that grpc-gateway should dial (usually gRPC server's DialAddress()).
func (p *GrpcProxy) SetGRPCEndpoint(endpoint string) *GrpcProxy {
	p.endpoint = endpoint
	return p
}

func (p *GrpcProxy) SetDialOptions(opts ...grpc.DialOption) *GrpcProxy {
	p.dialOptions = append([]grpc.DialOption(nil), opts...)
	p.dialOptions = append(p.dialOptions, grpc.WithChainUnaryInterceptor(TraceUnaryClientInterceptor()))
	return p
}

func (p *GrpcProxy) AddService(reg ServiceRegistry) *GrpcProxy {
	if reg != nil {
		p.services = append(p.services, reg)
	}
	return p
}

func (p *GrpcProxy) AddConnService(reg ConnRegistry) *GrpcProxy {
	if reg != nil {
		p.connRegs = append(p.connRegs, reg)
	}
	return p
}

func (p *GrpcProxy) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.onceStart.Do(func() {
			p.closeOnce.Do(func() { close(p.errCh) })
		})
		return nil
	}
	if p.endpoint == "" {
		return errors.New("grpc endpoint is not set (use SetGRPCEndpoint)")
	}
	if len(p.services) == 0 && len(p.connRegs) == 0 {
		return errors.New("no grpc-gateway services registered")
	}

	var startErr error
	p.onceStart.Do(func() {
		for _, reg := range p.services {
			if err := reg(ctx, p.mux, p.endpoint, p.dialOptions); err != nil {
				startErr = fmt.Errorf("register service: %w", err)
				return
			}
		}

		if len(p.connRegs) > 0 {
			conn, err := grpc.DialContext(ctx, p.endpoint, p.dialOptions...)
			if err != nil {
				startErr = fmt.Errorf("dial grpc %s: %w", p.endpoint, err)
				return
			}
			defer conn.Close()

			for _, reg := range p.connRegs {
				if err := reg(ctx, p.mux, conn); err != nil {
					startErr = fmt.Errorf("register conn service: %w", err)
					return
				}
			}
		}

		handler := otelhttp.NewHandler(
			http.Handler(p.mux),
			"grpc-gateway",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + r.URL.Path
			}),
		)
		if p.http == nil {
			p.http = &http.Server{
				ReadHeaderTimeout: 5 * time.Second,
			}
		}
		p.http.Addr = p.cfg.String()
		p.http.Handler = handler

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			err := p.http.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				p.errCh <- err
			}
			p.closeOnce.Do(func() { close(p.errCh) })
		}()
	})

	return startErr
}

func defaultProxyDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(TraceUnaryClientInterceptor()),
	}
}

func (p *GrpcProxy) Notify() <-chan error { return p.errCh }

func (p *GrpcProxy) Close() error {
	p.stopOnce.Do(func() {
		if p.http == nil {
			p.closeOnce.Do(func() { close(p.errCh) })
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = p.http.Shutdown(ctx)
		cancel()

		go func() {
			p.wg.Wait()
			p.closeOnce.Do(func() { close(p.errCh) })
		}()
	})
	return nil
}
