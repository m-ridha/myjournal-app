GOCMD ?= go
GOTEST ?= $(GOCMD) test ./...
GOLINT ?= golangci-lint

.PHONY: run generate generate-openapi generate-wire fmt lint test test-js

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

test-js:
	@if command -v node >/dev/null 2>&1; then node tests/editor.test.js && node tests/timezone.test.js; \
	elif [ -x /System/Library/Frameworks/JavaScriptCore.framework/Versions/A/Helpers/jsc ]; then \
	  /System/Library/Frameworks/JavaScriptCore.framework/Versions/A/Helpers/jsc tests/editor.test.js && \
	  /System/Library/Frameworks/JavaScriptCore.framework/Versions/A/Helpers/jsc tests/timezone.test.js; \
	else echo "Editor tests require Node.js or macOS JavaScriptCore (jsc)."; exit 1; fi

test: test-js
	$(GOTEST)
