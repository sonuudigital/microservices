FROM golang:1.25.3-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY shared/go.mod ./shared/
COPY gen/go.mod ./gen/
COPY cart-service/go.mod cart-service/go.sum ./cart-service/

WORKDIR /workspace/cart-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared
RUN go mod edit -replace github.com/sonuudigital/microservices/gen=../gen
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY gen/ ./gen/
COPY cart-service/ ./cart-service/

WORKDIR /workspace/cart-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared && \
    go mod edit -replace github.com/sonuudigital/microservices/gen=../gen && \
    go mod tidy

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/cart-service ./cmd/cart-service

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/cart-service /cart-service
COPY --from=builder /workspace/cart-service/internal/db/migrations /migrations

ENV PORT=8083

EXPOSE 8083

ENTRYPOINT ["/cart-service"]
