FROM golang:1.25.0-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY shared/go.mod shared/go.sum ./shared/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

WORKDIR /workspace/api-gateway
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY api-gateway/ ./api-gateway/

WORKDIR /workspace/api-gateway
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared && go mod tidy

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/api-gateway ./cmd/api-gateway

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/api-gateway /api-gateway

ENV PORT=8080

EXPOSE 8080

CMD ["/api-gateway"]
