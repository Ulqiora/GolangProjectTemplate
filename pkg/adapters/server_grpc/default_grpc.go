package server_grpc

import (
	"net"
	"sync"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	internal *grpc.Server
	listener net.Listener
	cfg      ServerConfig
}

func NewGrpcServer(cfg ServerConfig) (*GrpcServer, error) {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", cfg.String())
	if err != nil {
		return nil, err
	}
	return &GrpcServer{
		internal: grpcServer,
		listener: listener,
		cfg:      cfg,
	}, nil
}

func (s *GrpcServer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	go func() {
		if err := s.internal.Serve(s.listener); err != nil {
			panic(err)
		}
	}()
}

func (s *GrpcServer) Close() error {
	s.internal.Stop()
	return nil
}
