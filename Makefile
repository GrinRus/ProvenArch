GO ?= go
NPM ?= ./scripts/run-npm.sh
UI_DIR := ui
GO_FILES := $(shell find cmd internal -name '*.go' -type f 2>/dev/null)
RUNTIME ?= fake
REPO_NAME ?= primary-repo
DOCS_IMPORTS_PATH ?= ./docs/imports

.PHONY: bootstrap contracts test test-stress lint build run-backend run-ui quickstart-local

bootstrap:
	$(GO) mod tidy
	$(NPM) ci --prefix $(UI_DIR)

contracts:
	$(NPM) exec --yes --package=ajv-cli --package=ajv-formats --package=js-yaml -- bash ./scripts/validate-contracts.sh

test: contracts
	$(GO) test ./...
	python3 -m unittest discover -s scripts/tests -p '*_test.py'
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
	rm -rf $(UI_DIR)/dist $(UI_DIR)/node_modules/.vite
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

quickstart-local:
	@test -n "$(WORKSPACE)" || (echo "Set WORKSPACE=/abs/path/to/arch-workspace"; exit 1)
	@test -n "$(REPO_PATH)" || (echo "Set REPO_PATH=/abs/path/to/local/repo"; exit 1)
	$(GO) run ./cmd/acp init-workspace --workspace "$(WORKSPACE)" --repo-name "$(REPO_NAME)" --repo-path "$(REPO_PATH)" --docs-imports-path "$(DOCS_IMPORTS_PATH)" --force
	$(GO) run ./cmd/acp run --workspace "$(WORKSPACE)" --pipeline init --runtime "$(RUNTIME)" --non-interactive
	@echo "Next: $(GO) run ./cmd/acp serve --workspace \"$(WORKSPACE)\" --runtime \"$(RUNTIME)\""
