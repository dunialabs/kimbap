FROM golang:1.24.13-alpine AS builder
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X github.com/dunialabs/kimbap-core/internal/config.version=${VERSION}" -o /kimbap ./cmd/kimbap

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /kimbap .
RUN mkdir -p /data/kimbap && chown -R appuser:appgroup /app /data/kimbap
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/v1/health || exit 1
CMD ["./kimbap", "serve"]
