package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type healthCheckFunc func(ctx context.Context) error

func StartGRPCHealthCheckService(grpcServer *grpc.Server, service string, healthCheckFn healthCheckFunc) {
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		healthServer.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		if err := healthCheckFn(context.Background()); err == nil {
			healthServer.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}()
}
