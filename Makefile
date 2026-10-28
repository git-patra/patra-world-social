GOCACHE := $(CURDIR)/.cache/go-build
export GOCACHE

.PHONY: dev server lint typecheck test test-integration build check
dev:
	npm run dev
server:
	cd apps/server && go run ./cmd/server
lint:
	npm run lint
	cd apps/server && go vet ./...
typecheck:
	npm run typecheck
test:
	npm test
	cd apps/server && go test ./...
test-integration:
	cd apps/server && go test -tags=integration ./internal/auth
build:
	npm run build
	cd apps/server && go build -o ../../.cache/server ./cmd/server
check: lint typecheck test build

.PHONY: load-smoke
# Synthetic backend-only clients; each run creates and drops a random DB schema.
load-smoke:
	@set -a; if [ -z "$$DATABASE_URL" ] && [ -f .cache/database.env ]; then . .cache/database.env; fi; set +a; cd apps/server && go test -count=1 -race -tags=integration -run TestLoadSmoke -v ./internal/application
