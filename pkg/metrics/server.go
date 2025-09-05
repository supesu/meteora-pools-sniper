package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Server represents a Prometheus metrics HTTP server
type Server struct {
	server *http.Server
	logger logger.Logger
	port   int
}

// NewServer creates a new metrics server
func NewServer(port int, logger logger.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Server{
		server: server,
		logger: logger,
		port:   port,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	s.logger.WithField("port", s.port).Info("Starting metrics server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("Metrics server failed")
		}
	}()

	return nil
}

// Stop gracefully stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping metrics server")
	return s.server.Shutdown(ctx)
}

// GetPort returns the port the server is running on
func (s *Server) GetPort() int {
	return s.port
}
