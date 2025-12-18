SHELL = /bin/bash
VERSION ?= $(shell git describe --tags --always | cut -dv -f2)
BIN_DIR = bin

.PHONY: all
all: ## Build binaries
	go build -a -ldflags "-s -w -X main.version=$(VERSION)" -trimpath -o $(BIN_DIR)/ ./cmd/...

.PHONY: test
test: ## Exec unit tests
	go test -v ./...

.PHONY: lint
lint: ## Run linters
	go fmt ./...
	go vet ./...
	go tool staticcheck -f stylish ./...

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
