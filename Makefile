MAIN:=cmd/main.go

# Set image registry and image name
VERSION ?= dev
IMAGE_REPO ?= ghcr.io/nwneisen
IMAGE_TAG_BASE ?= $(IMAGE_REPO)/dex-http-server
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)

# LDFLAGS
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
	@docker build -t ${IMG} .

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

.PHONY: docker-clean
docker-clean: ## Clean out the docker image
	@docker image rm ${IMG}

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
