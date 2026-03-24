.PHONY: build run dev test clean deps lint vet

VERSION ?= 1.1.0
LDFLAGS ?= -X github.com/dunialabs/peta-core-go/internal/config.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/peta-core-go ./cmd/server

run: build
	./bin/peta-core-go

dev:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin/

deps:
	go mod tidy
	go mod download

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
