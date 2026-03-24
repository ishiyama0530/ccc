APP := claudecc
DOCKER_COMPOSE := docker compose run --build --rm dev
DOCKER_WORKDIR := /workspace
DOCKER_WORKSPACE_MOUNT := $(PWD):$(DOCKER_WORKDIR)
NPM_DOCKER_IMAGE := node:22-bookworm-slim
NPM_PACKAGE_NAME := claudecc
NPM_PACKAGE_DIR := dist/npm-package
NPM_REGISTRY := https://registry.npmjs.org
QUERY ?=
VERSION ?=
DIR ?=
HOST_GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_GOARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/arm64/arm64/')

.PHONY: run build test test-node lint release prepare-npm-package tidy

run: build
	@if [ -z "$(QUERY)" ]; then echo "QUERY is required" >&2; exit 1; fi
	@if [ -n "$(DIR)" ]; then \
		./bin/$(APP) -d "$(DIR)" "$(QUERY)"; \
	else \
		./bin/$(APP) "$(QUERY)"; \
	fi

build:
	@mkdir -p bin
	@$(DOCKER_COMPOSE) bash -c 'GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) CGO_ENABLED=0 go build -o bin/$(APP) ./cmd/ccc'

test:
	@$(DOCKER_COMPOSE) bash -c 'go test ./... && go test -race ./... && node --test packaging/npm/test/*.test.js'

test-node:
	@node --test packaging/npm/test/*.test.js

lint:
	@$(DOCKER_COMPOSE) bash -c 'golangci-lint run ./... && go vet ./...'

release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required" >&2; exit 1; fi
	@if ! printf '%s' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then echo "VERSION must match vX.Y.Z" >&2; exit 1; fi
	@if [ -z "$$GITHUB_TOKEN" ]; then echo "GITHUB_TOKEN is required" >&2; exit 1; fi
	@if [ -z "$$NPM_TOKEN" ]; then echo "NPM_TOKEN is required" >&2; exit 1; fi
	@command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
	@printf '==> preflight\n'
	@docker version >/dev/null 2>&1
	@docker run --rm \
		-v $(DOCKER_WORKSPACE_MOUNT) \
		-w $(DOCKER_WORKDIR) \
		$(NPM_DOCKER_IMAGE) \
		node packaging/npm/check-package-version.js --package "$(NPM_PACKAGE_NAME)" --tag "$(VERSION)" --registry "$(NPM_REGISTRY)"
	@printf '==> github release\n'
	@if ! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null; then git tag "$(VERSION)"; fi
	@git push origin "$(VERSION)"
	@docker run --rm \
		-e GITHUB_TOKEN \
		-v $(DOCKER_WORKSPACE_MOUNT) \
		-w $(DOCKER_WORKDIR) \
		goreleaser/goreleaser:v2.14.3 \
		release --clean
	@printf '==> npm package assembly\n'
	@rm -rf "$(NPM_PACKAGE_DIR)"
	@docker run --rm \
		-v $(DOCKER_WORKSPACE_MOUNT) \
		-w $(DOCKER_WORKDIR) \
		$(NPM_DOCKER_IMAGE) \
		node packaging/npm/prepare-package.js --tag "$(VERSION)" --out-dir "$(NPM_PACKAGE_DIR)"
	@printf '==> npm publish\n'
	@docker run --rm \
		-e NPM_TOKEN \
		-v $(DOCKER_WORKSPACE_MOUNT) \
		-w $(DOCKER_WORKDIR)/$(NPM_PACKAGE_DIR) \
		$(NPM_DOCKER_IMAGE) \
		sh -lc 'trap "rm -f .npmrc" EXIT; printf "//registry.npmjs.org/:_authToken=%s\n" "$$NPM_TOKEN" > .npmrc; npm publish --access public'

prepare-npm-package:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required" >&2; exit 1; fi
	@node packaging/npm/prepare-package.js --tag "$(VERSION)" --out-dir "$(NPM_PACKAGE_DIR)"

tidy:
	@$(DOCKER_COMPOSE) bash -c 'go mod tidy'
