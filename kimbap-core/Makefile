.PHONY: build run dev test clean deps lint vet install

VERSION ?= 1.1.0
LDFLAGS ?= -X github.com/dunialabs/kimbap-core/internal/config.version=$(VERSION)

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/kimbap ./cmd/kimbap

run: build
	./bin/kimbap daemon

dev:
	go run -ldflags "$(LDFLAGS)" ./cmd/kimbap daemon

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

install: build
	@echo "Installing kimbap to /usr/local/bin..."
	install -m 755 bin/kimbap /usr/local/bin/kimbap
	@rm -f /usr/local/bin/kb
	@ln -sf /usr/local/bin/kimbap /usr/local/bin/kb
	@echo "Installed: kimbap + kb alias -> /usr/local/bin/"
