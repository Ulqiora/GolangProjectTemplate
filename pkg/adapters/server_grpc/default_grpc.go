package server_grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type Registrar func(srv *grpc.Server)

type GrpcServer struct {
	internal *grpc.Server
	listener net.Listener
	cfg      ServerConfig

	registrars []Registrar

	errCh     chan error
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewGrpcServer(cfg ServerConfig, opts ...grpc.ServerOption) *GrpcServer {
	return &GrpcServer{
		internal: grpc.NewServer(opts...),
		cfg:      cfg,
		errCh:    make(chan error, 1),
	}
}

func (s *GrpcServer) SetListener(lis net.Listener) {
	s.listener = lis
}

func (s *GrpcServer) BaseServer() *grpc.Server { return s.internal }

func (s *GrpcServer) Register(registrars ...Registrar) {
	for _, r := range registrars {
		if r != nil {
			s.registrars = append(s.registrars, r)
		}
	}
}

func (s *GrpcServer) Start() error {
	var startErr error

	s.startOnce.Do(func() {
		for _, r := range s.registrars {
			r(s.internal)
		}

		if s.listener == nil {
			lis, err := net.Listen("tcp", s.cfg.String())
			if err != nil {
				startErr = fmt.Errorf("listen tcp %s: %w", s.cfg.String(), err)
				close(s.errCh)
				return
			}
			s.listener = lis
		}

		go func() {
			if err := s.internal.Serve(s.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				s.errCh <- err
			}
			close(s.errCh)
		}()
	})

	return startErr
}

func (s *GrpcServer) Close() error {
	s.stopOnce.Do(func() {
		s.internal.GracefulStop()
		if s.listener != nil {
			_ = s.listener.Close()
		}
	})
	return nil
}

func (s *GrpcServer) Notify() <-chan error { return s.errCh }

// Run starts the server and blocks until context cancellation or serve error.
func (s *GrpcServer) Run(ctx context.Context) error {
	if err := s.Start(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		_ = s.Close()
		return ctx.Err()
	case err := <-s.Notify():
		_ = s.Close()
		return err
	}
}
