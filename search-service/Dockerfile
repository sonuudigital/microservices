FROM golang:1.25.4-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace

COPY go.work go.work.sum ./
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
COPY cart-service/go.mod cart-service/go.sum ./cart-service/
COPY gen/go.mod gen/go.sum ./gen/
COPY notification-service/go.mod notification-service/go.sum ./notification-service/
COPY order-service/go.mod ./order-service/
COPY payment-service/go.mod ./payment-service/
COPY product-service/go.mod product-service/go.sum ./product-service/
COPY shared/go.mod shared/go.sum ./shared/
COPY tools/health-checker/go.mod ./tools/health-checker/
COPY user-service/go.mod user-service/go.sum ./user-service/
COPY search-service/go.mod search-service/go.sum ./search-service/

RUN --mount=type=cache,target=/go/pkg/mod go work sync
COPY . .
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/search-service ./search-service/cmd/search-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/search-service /search-service
ENTRYPOINT ["/search-service"]
