.PHONY: build build-all run demo test test-coverage lint fmt clean install deps release help

BINARY_NAME=grom
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/grom

build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/grom
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/grom
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/grom
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/grom

run: build
	./bin/$(BINARY_NAME)

demo: build
	./bin/$(BINARY_NAME) demo

deps:
	go mod download
	go mod tidy

test:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

fmt:
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

clean:
	rm -rf bin/ dist/
	rm -f coverage.out coverage.html

install:
	go install $(LDFLAGS) ./cmd/grom

# Release — tag-only; CI (goreleaser) is the sole publisher.
# Usage: make release V=0.1.0
release:
ifndef V
	$(error V is required. Usage: make release V=0.1.0)
endif
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory not clean."; exit 1; \
	fi
	@if [ "$$(git branch --show-current)" != "main" ]; then \
		echo "❌ Must be on main branch."; exit 1; \
	fi
	git tag v$(V)
	git push origin v$(V)
	@echo "✅ Tag v$(V) pushed. CI publishes the release."

help:
	@echo "grom Makefile:"
	@echo "  make build          Build the binary"
	@echo "  make build-all      Build for all platforms"
	@echo "  make demo           Build and render the demo gallery"
	@echo "  make test           Run tests (-race)"
	@echo "  make lint           Run golangci-lint"
	@echo "  make fmt            Format code"
	@echo "  make install        Install to GOPATH/bin"
	@echo "  make release V=x.y.z  Tag and push a release"
