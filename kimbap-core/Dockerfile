FROM golang:1.24.13-alpine AS builder
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X github.com/dunialabs/kimbap-core/internal/config.version=${VERSION}" -o /kimbap-core ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /kimbap-core .
COPY --from=builder /app/internal/oauth/views ./internal/oauth/views
RUN chown -R appuser:appgroup /app
USER appuser
EXPOSE 3002
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3002/health || exit 1
CMD ["./kimbap-core"]
