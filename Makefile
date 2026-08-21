# Everything CI runs, runnable locally with the same command.
# `make check` is the gate: if it passes here it passes there.

GOLANGCI_VERSION ?= v2.12.2   # keep in step with .github/workflows/ci.yml

.PHONY: check build test fmt vet lint tidy tools clean preview icons font release release-plan

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

font: ## install the glyphs the icons need
	@# Symbols-only, so a terminal falls back to it per glyph and nobody has to
	@# give up the text font they already like.
	brew install --cask font-symbols-only-nerd-font

icons: ## show the candidate icons, to be judged by eye
	go run ./cmd/icons

preview: ## render the theme, both light and dark
	go run ./cmd/preview

release-plan: ## show the tag a release would create, and why
	@scripts/release.sh plan

# Depends on check: a tag is what another repo pins, so it names a commit that
# built and passed. LEVEL overrides the bump the commit subjects imply, and NOTE
# adds the phrase after the number.
release: check ## tag the next version and push it (LEVEL=, NOTE= override)
	@scripts/release.sh tag $(LEVEL)

clean:
	go clean -cache -testcache
	rm -rf bin dist

help: ## list targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | expand -t22
