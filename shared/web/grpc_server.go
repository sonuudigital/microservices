package web

import (
	"context"
	"net"

	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/grpc"
)

func StartGRPCServerAndWaitForShutdown(ctx context.Context, grpcServer *grpc.Server, lis net.Listener, logger logs.Logger) error {
	errChan := make(chan error, 1)

	go func() {
		logger.Info("gRPC server listening", "port", lis.Addr().(*net.TCPAddr).Port)
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down gRPC server")
		grpcServer.GracefulStop()
		logger.Info("gRPC server stopped gracefully")
		return ctx.Err()
	case err := <-errChan:
		logger.Error("gRPC server failed", "error", err)
		return err
	}
}
