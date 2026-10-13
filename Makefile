GOCMD ?= go
GOTEST ?= $(GOCMD) test ./...
GOLINT ?= golangci-lint

.PHONY: run generate generate-openapi generate-wire fmt lint test test-js test-architecture

all: generate fmt lint test run

run:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	$(GOCMD) run ./cmd/app

generate: generate-openapi \
	generate-wire	

generate-openapi:
	@mkdir -p internal/driver/web/api
	@oapi-codegen -generate "types,chi-server" -package api -o internal/driver/web/api/openapi.gen.go docs/api/openapi.yml

generate-wire:
	wire ./...

fmt:
	$(GOLINT) fmt ./...

lint:
	$(GOLINT) run

test: test-js
	$(GOTEST)

test-js:
	@if command -v node >/dev/null 2>&1; then node tests/editor.test.js && node tests/timezone.test.js && node tests/browsing.test.js && node tests/theme.test.js && node tests/benchmark.test.js && node tests/language.test.js; \
	elif [ -x /System/Library/Frameworks/JavaScriptCore.framework/Versions/A/Helpers/jsc ]; then \
	  for suite in editor timezone browsing theme benchmark language; do \
	    result=$$(/System/Library/Frameworks/JavaScriptCore.framework/Versions/A/Helpers/jsc tests/$$suite.test.js) || exit 1; \
	    printf '%s\n' "$$result"; \
	    case "$$result" in *"PASS $$suite"*) ;; *) exit 1 ;; esac; \
	  done; \
	else echo "Browser interaction tests require Node.js or macOS JavaScriptCore (jsc)."; exit 1; fi

# Run separately: this temporarily adds forbidden imports and invokes lint.
test-architecture:
	sh scripts/shell/check-architecture.sh
