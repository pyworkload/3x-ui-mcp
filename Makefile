BINARY_NAME := xui-mcp
CMD_PATH := ./cmd/xui-mcp
BUILD_DIR := bin

# Version info embedded at build time
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: build run clean test lint fmt vet tidy help

## build: Compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

## run: Build and run the server
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) dist

## test: Run all tests
test:
	go test -v -race ./...

## cover: Run tests with coverage report
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run Go vet and staticcheck
lint: vet
	@which staticcheck > /dev/null 2>&1 || echo "install: go install honnef.co/go/tools/cmd/staticcheck@latest"
	staticcheck ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format all Go files
fmt:
	gofmt -s -w .

## tidy: Clean up go.mod and go.sum
tidy:
	go mod tidy

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
