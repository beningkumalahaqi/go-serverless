.PHONY: help run test build dev preview deploy portability

MODULE := $(shell go list -m)

help: ## list targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

run: ## local server on :8080
	go run ./cmd/server

test: ## vet and run the race-enabled tests
	go vet ./... && go test ./... -race -count=1

build: ## standalone binary for Docker and other hosts
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server

dev: ## Vercel emulation (needs the vercel CLI)
	vercel dev

preview: ## preview deployment
	vercel deploy

deploy: ## production deployment
	vercel deploy --prod

portability: ## core must stay stdlib-only and Vercel-free
	@! go list -deps ./api/... ./core/... ./docs/... | grep -v '^$(MODULE)' | grep -E '^[^/]+\.[^/]+/' || (echo "third-party dependency found in core" && exit 1)
	@! grep -rIl --include='*.go' -i 'vercel' . | grep -v '^\./api/' | grep -v '^\./\.vercel/' || (echo "vercel reference in Go code" && exit 1)
