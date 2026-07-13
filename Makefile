BIN_DIR   := bin
CLI       := $(BIN_DIR)/qrgo
WASM_OUT  := web/qr.wasm
WASM_EXEC := web/src/vendor/wasm_exec.js
GOROOT    := $(shell go env GOROOT)

.PHONY: all build wasm serve web-build web-test web-smoke test race vuln cover vet fmt fmt-check check install clean help

all: build wasm ## Build the CLI and the wasm bundle

build: ## Build the CLI into bin/qrgo
	go build -o $(CLI) ./cmd/qrgo

install: ## Install the CLI into GOPATH/bin
	go install ./cmd/qrgo

# wasm-opt (binaryen) trims ~7% off the raw binary (mostly parse/instantiate
# time; the gzipped wire size barely moves). Echo which path ran, because a silent
# branch here once made two builds incomparable.
wasm: ## Build the wasm module (stripped + wasm-opt if installed) and copy wasm_exec.js into web/
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $(WASM_OUT) ./cmd/wasm
	@if command -v wasm-opt >/dev/null; then \
		echo "wasm-opt -Oz ($$(wasm-opt --version))"; \
		wasm-opt -Oz --enable-bulk-memory --enable-nontrapping-float-to-int --enable-sign-ext --enable-mutable-globals $(WASM_OUT) -o $(WASM_OUT); \
	else \
		echo "wasm-opt not found (brew install binaryen), shipping unoptimized wasm"; \
	fi
	@echo "$(WASM_OUT): $$(wc -c < $(WASM_OUT) | tr -d ' ') bytes ($$(gzip -9 -c $(WASM_OUT) | wc -c | tr -d ' ') gzipped)"
	@mkdir -p $(dir $(WASM_EXEC))
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" $(WASM_EXEC)

serve: wasm ## Build wasm and start the web dev server (bun, with HMR)
	cd web && bun index.html

web-check: wasm ## Typecheck the web app
	cd web && bun run typecheck

web-test: ## Run Bun unit tests
	cd web && bun test src

web-smoke: wasm ## Run the Chromium browser smoke tests
	cd web && bun run smoke

web-build: wasm ## Build the production web bundle into web/dist
	rm -rf web/dist
	cd web && bun build index.html --outdir dist --minify
	cp web/_headers web/robots.txt web/llms.txt web/dist/
	cp -R web/fonts web/dist/
	@if [ -f web/social-card.png ]; then cp web/social-card.png web/dist/; else echo "warning: web/social-card.png missing (og:image will 404)"; fi
	@echo "dist: $$(du -sh web/dist | cut -f1)"

test: ## Run all tests
	go test ./...

race: ## Run all tests with the race detector
	go test -count=1 -race ./...

vuln: ## Scan Go dependencies with govulncheck
	govulncheck ./...

cover: ## Run tests with coverage and open the report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -w .

fmt-check: ## Fail if any file is not gofmt-ed (CI check)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-ed:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

check: fmt-check vet build race ## Core Go checks (CI also runs vuln and web gates)

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out $(WASM_OUT)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
