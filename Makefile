# Everything CI runs, runnable locally with the same command.
# `make check` is the gate: if it passes here it passes there.

GOLANGCI_VERSION ?= v2.12.2   # keep in step with .github/workflows/ci.yml

.PHONY: check build test fmt vet lint tidy tools clean

check: fmt vet lint test ## everything CI runs

build: ## compile every package
	go build ./...

test: ## run the tests with the race detector
	go test -race ./...

fmt: ## rewrite formatting, then fail if anything changed
	gofmt -l -w .
	@test -z "$$(gofmt -l .)" || { echo "gofmt changed files; commit them"; exit 1; }

vet: ## the compiler's own correctness checks
	go vet ./...

lint: ## golangci-lint, configured in .golangci.yml
	@command -v golangci-lint >/dev/null || { \
		echo "golangci-lint missing — run: make tools"; exit 1; }
	golangci-lint run

tidy: ## drop unused requirements, verify the checksums
	go mod tidy
	go mod verify

tools: ## install the pinned linter
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

clean:
	go clean -cache -testcache
	rm -rf bin dist

help: ## list targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | expand -t22
