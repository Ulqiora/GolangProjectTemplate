package server_grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

type AppConfig struct {
	GRPC  ServerConfig `yaml:"grpc"`
	Proxy ProxyConfig  `yaml:"proxy"`
}

// App runs gRPC server with optional grpc-gateway proxy.
type App struct {
	grpc  *GrpcServer
	proxy *GrpcProxy

	errCh chan error
	wg    sync.WaitGroup
	once  sync.Once
}

func NewApp(cfg AppConfig, grpcOpts ...grpc.ServerOption) (*App, error) {
	app := &App{
		grpc:  NewGrpcServer(cfg.GRPC, grpcOpts...),
		errCh: make(chan error, 2),
	}

	if cfg.Proxy.Enabled {
		app.proxy = NewProxy(cfg.Proxy).SetGRPCEndpoint(cfg.GRPC.DialAddress())
	}

	return app, nil
}

func (a *App) GRPC() *GrpcServer { return a.grpc }

func (a *App) Proxy() *GrpcProxy { return a.proxy }

func (a *App) Start(ctx context.Context) error {
	if a.grpc == nil {
		return errors.New("grpc server is nil")
	}

	if err := a.grpc.Start(); err != nil {
		return fmt.Errorf("start grpc: %w", err)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := <-a.grpc.Notify(); err != nil {
			a.errCh <- fmt.Errorf("grpc: %w", err)
		}
	}()

	if a.proxy != nil {
		if err := a.proxy.Start(ctx); err != nil {
			_ = a.grpc.Close()
			return fmt.Errorf("start grpc-proxy: %w", err)
		}

		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			if err := <-a.proxy.Notify(); err != nil {
				a.errCh <- fmt.Errorf("grpc-proxy: %w", err)
			}
		}()
	}

	return nil
}

func (a *App) Notify() <-chan error { return a.errCh }

func (a *App) Close() error {
	if a.proxy != nil {
		_ = a.proxy.Close()
	}
	if a.grpc != nil {
		_ = a.grpc.Close()
	}
	go func() {
		a.wg.Wait()
		a.once.Do(func() { close(a.errCh) })
	}()
	return nil
}
