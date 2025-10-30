FROM golang:1.25.3-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY shared/go.mod shared/go.sum ./shared/
COPY gen/go.mod gen/go.sum ./gen/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

WORKDIR /workspace/api-gateway
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared
RUN go mod edit -replace github.com/sonuudigital/microservices/gen=../gen
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY gen/ ./gen/
COPY api-gateway/ ./api-gateway/

WORKDIR /workspace/api-gateway
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared && \
    go mod edit -replace github.com/sonuudigital/microservices/gen=../gen && \
    go mod tidy

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.20

RUN apk --no-cache add curl ca-certificates

COPY --from=builder /out/api-gateway /api-gateway

ENV PORT=8080

EXPOSE 8080

RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
USER nonroot

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1

CMD ["/api-gateway"]
