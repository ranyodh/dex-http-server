MAIN:=cmd/dex-http-server/main.go

# LDFLAGS
VERSION:=dev
# VERSION := $(shell git tag --sort=committerdate | tail -1)
COMMIT := $(shell git rev-parse HEAD)
DATE := $(shell date -u '+%Y-%m-%d')
LDFLAGS=-ldflags \
				" \
				-X github.com/nwneisen/dex-http-server/cmd/main.version=${VERSION} \
				-X github.com/nwneisen/dex-http-server/cmd/main.commit=${COMMIT} \
				-X github.com/nwneisen/dex-http-server/cmd/main.date=${DATE} \
				"

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: all
all: fmt vet build ## Do all the things

.PHONY: print-%
print-%:
	@echo $($*)

##@ Development

.PHONY: build
build: ## Build the binary
	@go build ${LDFLAGS} -o bin/dex-http-server ${MAIN}

.PHONY: run
run: ## Run the binary
	@go run ${LDFLAGS} ${MAIN}

.PHONY: clean
clean: ## Clean out the binary
	@rm -rf bin

##@ Docker

.PHONY: up
up: ## Run the project in docker containers
	@docker compose up --build

.PHONY: docker-build
docker-build: ## Build the docker image
	@docker build -t ghcr.io/nwneisen/dex-http-server:${VERSION} .

.PHONY: docker-clean
docker-clean: ## Clean out the docker image
	@docker image rm ghcr.io/nwneisen/dex-http-server:${VERSION}

##@ Testing

.PHONY: test
test: fmt vet static ## Run all tests

.PHONY: fmt
fmt: ## Run go fmt against code.
	@go fmt ${MAIN}

.PHONY: vet
vet: ## Run go vet against code.
	@go vet ${MAIN}

.PHONY: static
static: ## Run staticcheck against code.
	@staticcheck -checks "all" ${MAIN}

##@ Dependencies

.PHONY: deps
deps: staticcheck ## Install all of hte needed dependencies

.PHONY: ginkgo
ginkgo: ## Install staticcheck - doesn't work
	@go install github.com/onsi/ginkgo/v2/ginkgo

.PHONY: staticcheck
staticcheck: ## Install staticcheck
	@go install honnef.co/go/tools/cmd/staticcheck@latest
