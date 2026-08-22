# Development entry points for the Go side of the repository.
#
# GOTMPDIR is overridable and defaults outside /tmp because some hardened hosts
# mount /tmp with `noexec`, which prevents `go test` from running the compiled
# test binaries.
GOTMPDIR ?= $(HOME)/.cache/go-tmp
export GOTMPDIR

MODULE  := github.com/leohteixeira/opensource-project-intelligence
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: help
help: ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

$(GOTMPDIR):
	@mkdir -p $(GOTMPDIR)

.PHONY: build
build: ## Build every binary
	go build ./...

.PHONY: fmt
fmt: ## Format the Go sources
	gofmt -w $(GOFILES)

.PHONY: fmt-check
fmt-check: ## Fail when a Go source is not formatted
	@unformatted="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "these files are not formatted:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: $(GOTMPDIR) ## Run the unit tests
	go test ./...

.PHONY: test-race
test-race: $(GOTMPDIR) ## Run the tests with the race detector
	go test -race ./...

.PHONY: run-api
run-api: ## Run the API
	go run ./cmd/api

.PHONY: run-worker
run-worker: ## Run the worker
	go run ./cmd/worker

.PHONY: migrate
migrate: ## Apply the SQL migrations
	./scripts/migrate.sh up

.PHONY: migrate-down
migrate-down: ## Revert the latest SQL migration
	./scripts/migrate.sh down

.PHONY: generate
generate: ## Regenerate HTTP, SQL, and TypeScript adapters
	./scripts/generate.sh

.PHONY: generate-check
generate-check: ## Fail when reviewed sources or generated adapters drift
	./scripts/check-generated.sh

.PHONY: test-integration
test-integration: $(GOTMPDIR) ## Run tests against the configured real services
	@test -n "$(OPI_INTEGRATION_DATABASE_URL)" || \
		(echo "OPI_INTEGRATION_DATABASE_URL is required"; exit 1)
	go test -tags=integration ./...

.PHONY: backup
backup: ## Back up PostgreSQL and the canonical object manifest
	./scripts/backup.sh

.PHONY: restore
restore: ## Restore PostgreSQL and verify the backup manifest
	./scripts/restore.sh

.PHONY: check
check: generate-check fmt-check vet test-race build ## Everything CI runs for the Go side
