FROM golang:1.25.3-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY shared/go.mod ./shared/
COPY gen/go.mod ./gen/
COPY product-service/go.mod product-service/go.sum ./product-service/

WORKDIR /workspace/product-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared
RUN go mod edit -replace github.com/sonuudigital/microservices/gen=../gen
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY gen/ ./gen/
COPY product-service/ ./product-service/

WORKDIR /workspace/product-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared && \
    go mod edit -replace github.com/sonuudigital/microservices/gen=../gen && \
    go mod tidy

RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/product-service ./cmd/product-service

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/product-service /product-service
COPY --from=builder /go/bin/grpc-health-probe /grpc_health_probe
COPY --from=builder /workspace/product-service/internal/database/migrations /migrations

ENV PORT=8082

EXPOSE 8082

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/grpc_health_probe", "-addr=:8082", "-service=product-service"]

ENTRYPOINT ["/product-service"]
