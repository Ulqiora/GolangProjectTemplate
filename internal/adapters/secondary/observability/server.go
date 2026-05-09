package observability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"GolangTemplateProject/internal/config"
	prommodule "GolangTemplateProject/pkg/prometheus"
)

type Server struct {
	httpServer *http.Server
}

func NewMetricsServer(cfg config.HTTPServer) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prommodule.Handler())

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func NewProfilerServer(cfg config.HTTPServer) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	go func() {
		_ = s.httpServer.ListenAndServe()
	}()
	return nil
}

func (s *Server) Close() error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
