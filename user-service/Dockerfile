FROM golang:1.25.0-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY shared/go.mod ./shared/
COPY user-service/go.mod user-service/go.sum ./user-service/

WORKDIR /workspace/user-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY user-service/ ./user-service/

WORKDIR /workspace/user-service
RUN go mod edit -replace github.com/sonuudigital/microservices/shared=../shared && go mod tidy

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" \
    -o /out/user-service ./cmd/user-service

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/user-service /user-service
COPY --from=builder /workspace/user-service/internal/migrations /migrations

ENV PORT=8081

EXPOSE 8081

CMD ["/user-service"]