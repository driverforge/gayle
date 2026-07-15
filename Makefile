# driverforge/gayle Makefile

BINARY      := gayle
MODULE      := github.com/driverforge/gayle
BUILDINFO   := $(MODULE)/internal/buildinfo
DIST        := dist

# Version metadata stamped into the binary via -ldflags.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(BUILDINFO).Version=$(VERSION) \
	-X $(BUILDINFO).Commit=$(COMMIT) \
	-X $(BUILDINFO).Date=$(DATE)

# Platforms for `make release` (GOOS/GOARCH pairs).
PLATFORMS := darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/arm64 windows/amd64

# Install prefix for `make install-bin` (the end-user layout, like the curl installer).
PREFIX ?= /usr/local

.DEFAULT_GOAL := build

.PHONY: build
build: ## Build the gayle binary for the host platform
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/gayle

.PHONY: install
install: ## Install gayle into $GOBIN (or ~/go/bin) via go install
	go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/gayle

.PHONY: install-bin
install-bin: build ## Install the built binary to $(PREFIX)/bin (end-user layout; may need sudo)
	install -d "$(PREFIX)/bin"
	install -m 0755 "$(BINARY)" "$(PREFIX)/bin/$(BINARY)"

.PHONY: run
run: ## Build and run (use ARGS="list -s dev")
	go run -ldflags '$(LDFLAGS)' ./cmd/gayle $(ARGS)

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: cover
cover: ## Run tests with a coverage summary
	go test -cover ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go sources
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any Go source is not gofmt-clean
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not gofmt-clean:"; echo "$$out"; exit 1; fi

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	go mod tidy

.PHONY: check
check: fmt-check vet test ## Run fmt-check, vet, and tests

.PHONY: release
release: ## Cross-compile binaries into $(DIST)/ for all PLATFORMS
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=$(DIST)/$(BINARY)-$$os-$$arch; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build -trimpath -ldflags '$(LDFLAGS)' -o $$out ./cmd/gayle || exit 1; \
	done

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BINARY) $(DIST)

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
