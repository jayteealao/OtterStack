.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -tags=integration ./...

.PHONY: test-integration-verbose
test-integration-verbose: ## Run integration tests with verbose output
	go test -v -tags=integration ./...

.PHONY: test-all
test-all: test test-race test-integration ## Run all tests (unit, race detector, integration)

.PHONY: build
build: ## Build the binary
	go build -o otterstack.exe .

.PHONY: clean
clean: ## Clean build artifacts
	rm -f otterstack.exe

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: vet ## Run linters
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod tidy

.PHONY: ci
ci: fmt vet test test-race ## Run CI checks locally (format, vet, test, race detector)
