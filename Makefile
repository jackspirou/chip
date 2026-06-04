# chip — a toy systems scripting language.

GO  ?= go
BIN ?= chip

GOFUMPT      := mvdan.cc/gofumpt@latest
GOLANGCILINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
GOVULNCHECK  := golang.org/x/vuln/cmd/govulncheck@latest

.DEFAULT_GOAL := help
.PHONY: help build run test vet fmt lint vuln tidy check clean

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-8s %s\n", $$1, $$2}'

build: ## Compile the chip binary
	$(GO) build -o $(BIN) ./cmd/chip

run: build ## Build and run the gcd example
	./$(BIN) test/gcd_main.chp

test: ## Run tests with the race detector
	$(GO) test -race -shuffle=on ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format the code with gofumpt
	$(GO) run $(GOFUMPT) -w .

lint: ## Run golangci-lint
	$(GO) run $(GOLANGCILINT) run ./...

vuln: ## Scan for known vulnerabilities (govulncheck)
	$(GO) run $(GOVULNCHECK) ./...

tidy: ## Tidy go.mod
	$(GO) mod tidy

check: test lint vuln ## Run tests, linters, and the vuln scan (CI parity)

clean: ## Remove build artifacts
	rm -f $(BIN)
	$(GO) clean
