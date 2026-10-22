# Everything CI runs, runnable locally with the same command.
# `make check` is the gate: if it passes here it passes there.

GOLANGCI_VERSION ?= v2.12.2   # keep in step with .github/workflows/ci.yml

.PHONY: check build test fmt vet lint tidy tools clean preview icons font release release-plan

check: fmt vet lint test ## everything CI runs

build: ## compile every package
	go build ./...

test: ## run the tests with the race detector, then again as CI's shell
	go test -race ./...
# A second run of the packages that drive a shell, against dash.
#
# /bin/sh is bash-ish on a Mac and dash on the machines CI uses, so a test
# written against one passes here and fails there — which is how two pushes in a
# row went red on something a local run could never have seen. Skipped where
# dash is not installed rather than failing: it is a second opinion, not a
# dependency.
	@if command -v dash >/dev/null; then \
		echo "go test ./grid ./pane  (BELP_TEST_SH=dash)"; \
		BELP_TEST_SH=$$(command -v dash) go test -count=1 ./grid/ ./pane/; \
	else \
		echo "dash not installed — skipping the second shell"; \
	fi

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

font: ## install the patched font the icons need
	@# The terminal's own font has to carry the glyphs. A symbols-only fallback
	@# does not help: a private-use codepoint has no script, so nothing can infer
	@# a fallback font from it. Installing is half the job — the profile has to
	@# name it too.
	brew install --cask font-jetbrains-mono-nerd-font
	@echo
	@echo "Now set your terminal profile's font to JetBrainsMono Nerd Font,"
	@echo "then check with: make icons"

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
