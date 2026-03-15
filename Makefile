APP := ccc
DOCKER_COMPOSE := docker compose run --build --rm dev
QUERY ?=
VERSION ?=
TAP_REPO ?=
DIR ?=
HOST_GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_GOARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/arm64/arm64/')
TAP_OWNER := $(shell printf '%s' "$(TAP_REPO)" | cut -d/ -f1)
TAP_NAME := $(shell printf '%s' "$(TAP_REPO)" | cut -d/ -f2)

.PHONY: run build test lint release tidy

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
	@$(DOCKER_COMPOSE) bash -c 'go test ./... && go test -race ./...'

lint:
	@$(DOCKER_COMPOSE) bash -c 'golangci-lint run ./... && go vet ./...'

release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required" >&2; exit 1; fi
	@if [ -z "$(TAP_REPO)" ]; then echo "TAP_REPO is required" >&2; exit 1; fi
	@if [ -z "$(TAP_OWNER)" ] || [ -z "$(TAP_NAME)" ]; then echo "TAP_REPO must be owner/name" >&2; exit 1; fi
	@if ! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null; then git tag "$(VERSION)"; fi
	@git push origin "$(VERSION)"
	@docker run --rm \
		-e GITHUB_TOKEN \
		-e TAP_OWNER=$(TAP_OWNER) \
		-e TAP_NAME=$(TAP_NAME) \
		-v $(PWD):/workspace \
		-w /workspace \
		goreleaser/goreleaser:v2.14.3 \
		release --clean

tidy:
	@$(DOCKER_COMPOSE) bash -c 'go mod tidy'
