GOCMD ?= go
GOTEST ?= $(GOCMD) test ./...
GOLINT ?= golangci-lint

.PHONY: run generate generate-openapi generate-wire fmt lint test

run:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	$(GOCMD) run ./cmd/app

generate: generate-openapi \
	generate-wire	

generate-openapi:
	@mkdir -p internal/web/api
	@oapi-codegen -generate "types,chi-server" -package api -o internal/web/api/openapi.gen.go docs/api/openapi.yml

generate-wire:
	wire ./...

fmt:
	$(GOCMD) fmt ./...

lint:
	$(GOLINT) run

test:
	$(GOTEST)
