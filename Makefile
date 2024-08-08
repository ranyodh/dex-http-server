MAIN:=cmd/dex-http-server/main.go

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: all
all: fmt vet build ## Do all the things

##@ Development

.PHONY: build
build: ## Build the binary
	@go build -o bin/dex-http-server ${MAIN}

.PHONY: run
run: ## Run the binary
	@go run ${MAIN}

.PHONY: clean
clean: ## Clean out the binary
	@rm -rf bin

##@ Docker

.PHONY: up
up: ## Run the project in docker containers
	@docker compose up

.PHONY: docker-build
docker-build: ## Build the docker image
	@docker build -t dex-http-server .

.PHONY: docker-clean
docker-clean: ## Clean out the docker image
	@docker image rm dex-http-server

##@ Testing

.PHONY: fmt
fmt: ## Run go fmt against code.
	@go fmt ${MAIN}

.PHONY: vet
vet: ## Run go vet against code.
	@go vet ${MAIN}

.PHONY: static
static: ## Run staticcheck against code.
	@staticcheck ${MAIN}

##@ Dependencies

.PHONY: deps
deps: staticcheck ## Install all of hte needed dependencies

.PHONY: staticcheck
staticcheck: ## Install staticcheck
	@go install honnef.co/go/tools/cmd/staticcheck@latest
