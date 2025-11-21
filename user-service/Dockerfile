FROM golang:1.25.4-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace

COPY go.work go.work.sum ./
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
COPY cart-service/go.mod cart-service/go.sum ./cart-service/
COPY gen/go.mod gen/go.sum ./gen/
COPY notification-service/go.mod ./notification-service/
COPY order-service/go.mod ./order-service/
COPY payment-service/go.mod ./payment-service/
COPY product-service/go.mod product-service/go.sum ./product-service/
COPY shared/go.mod shared/go.sum ./shared/
COPY tools/health-checker/go.mod ./tools/health-checker/
COPY user-service/go.mod user-service/go.sum ./user-service/
COPY search-service/go.mod search-service/go.sum ./search-service/

RUN --mount=type=cache,target=/go/pkg/mod go work sync

COPY . .

RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/user-service ./user-service/cmd/user-service

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/user-service /user-service
COPY --from=builder /go/bin/grpc-health-probe /grpc_health_probe
COPY --from=builder /workspace/user-service/internal/migrations /migrations

ENV PORT=8081

EXPOSE 8081

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/grpc_health_probe", "-addr=:8081", "-service=user-service"]

ENTRYPOINT ["/user-service"]