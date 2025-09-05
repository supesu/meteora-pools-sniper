package grpc

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Middleware handles gRPC middleware setup
type Middleware struct{}

// NewMiddleware creates a new middleware instance
func NewMiddleware() *Middleware {
	return &Middleware{}
}

// Setup creates gRPC server options with middleware
func (m *Middleware) Setup(logger *logrus.Logger) []grpc.ServerOption {
	logrusEntry := logrus.NewEntry(logger)

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_logrus.StreamServerInterceptor(logrusEntry),
			grpc_recovery.StreamServerInterceptor(),
		)),
	}
}
