package web

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sonuudigital/microservices/shared/logs"
	"google.golang.org/grpc"
)

func StartGRPCServerAndWaitForShutdown(grpcServer *grpc.Server, lis net.Listener, logger logs.Logger) {
	go func() {
		logger.Info("gRPC server listening", "port", lis.Addr().(*net.TCPAddr).Port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("shutting down gRPC server")
	grpcServer.GracefulStop()
	logger.Info("shutdown complete")
}
