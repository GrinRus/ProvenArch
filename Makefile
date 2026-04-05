GO ?= go
NPM ?= npm
UI_DIR := ui
GO_FILES := $(shell find cmd internal -name '*.go' -type f 2>/dev/null)

.PHONY: bootstrap contracts test test-stress lint build run-backend run-ui

bootstrap:
	$(GO) mod tidy
	$(NPM) ci --prefix $(UI_DIR)

contracts:
	$(NPM) exec --yes --package=ajv-cli --package=ajv-formats --package=js-yaml -- bash ./scripts/validate-contracts.sh

test: contracts
	$(GO) test ./...
	$(NPM) run test --prefix $(UI_DIR) -- --run

test-stress:
	$(GO) test ./internal/orchestrator -run TestStartAsyncRunRejectsWhenPendingOutsideDebounceWindow -count=30

lint:
	@fmt_files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$fmt_files" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$fmt_files"; \
		exit 1; \
	fi
	$(NPM) run typecheck --prefix $(UI_DIR)

build:
	$(NPM) run build --prefix $(UI_DIR)
	rm -rf internal/api/ui_dist/assets internal/api/ui_dist/index.html
	mkdir -p internal/api/ui_dist/assets
	cp -R ui/dist/assets/* internal/api/ui_dist/assets/
	cp ui/dist/index.html internal/api/ui_dist/index.html
	mkdir -p ./bin
	$(GO) build -o ./bin/acp ./cmd/acp

run-backend:
	@test -n "$(WORKSPACE)" || (echo "Set WORKSPACE=/abs/path/to/arch-workspace"; exit 1)
	$(GO) run ./cmd/acp serve --workspace "$(WORKSPACE)"

run-ui:
	$(NPM) run dev --prefix $(UI_DIR)
