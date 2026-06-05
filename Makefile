# chip — a toy systems scripting language.

GO  ?= go
BIN ?= chip

# Tool versions. Keep GOLANGCI_VERSION in sync with the golangci-lint-action
# `version:` in .github/workflows/ci.yml so `make lint` and CI run the same
# linter at the same version against the same .golangci.yml.
GOLANGCI_VERSION ?= v2.12.2
GOFUMPT      := mvdan.cc/gofumpt@latest
GOLANGCILINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
GOVULNCHECK  := golang.org/x/vuln/cmd/govulncheck@latest

.DEFAULT_GOAL := help
.PHONY: help build bin run test vet fmt lint vuln tidy check clean

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-8s %s\n", $$1, $$2}'

build: ## Compile every package (CI parity)
	$(GO) build ./...

bin: ## Build the chip binary
	$(GO) build -o $(BIN) ./cmd/chip

run: bin ## Build and run the gcd example
	./$(BIN) examples/gcd.chp

test: ## Run tests with the race detector (CI parity)
	$(GO) test -race -shuffle=on ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format the code with gofumpt
	$(GO) run $(GOFUMPT) -w .

lint: ## Run golangci-lint, pinned to match CI
	$(GO) run $(GOLANGCILINT) run ./...

vuln: ## Scan for known vulnerabilities (govulncheck)
	$(GO) run $(GOVULNCHECK) ./...

tidy: ## Tidy go.mod
	$(GO) mod tidy

check: build test lint vuln ## Run the full CI-equivalent gate

clean: ## Remove build artifacts
	rm -f $(BIN)
	$(GO) clean
