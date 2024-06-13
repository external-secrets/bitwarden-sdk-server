NAME=bitwarden-sdk-server

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR := bin

# VERSION defines the project version for the bundle.
VERSION ?= 0.0.1

UNAME ?= $(shell uname|tr '[:upper:]' '[:lower:]')

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


# List the GOOS and GOARCH to build
GO_LDFLAGS_STATIC="-s -w $(CTIMEVAR) -extldflags -static"
GOLANGCI_LINT_VERSION ?= v1.57.2

.DEFAULT_GOAL := help

##@ Build

build: ## Builds binaries
	CGO_LDFLAGS="-framework CoreFoundation" CGO_ENABLED=1 go build -ldflags='-s -w' -o $(LOCALBIN)/$(NAME) main.go

build-docker: ## Builds binaries
	CC=musl-gcc CGO_LDFLAGS="-lm" CGO_ENABLED=1 go build -a -ldflags '-linkmode external -extldflags "-static -Wl,-unresolved-symbols=ignore-all"' -o $(LOCALBIN)/$(NAME) main.go

##@ Testing

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

clean: ## Runs go clean
	go clean -i

test: ## Test the project
	go test ./... -coverprofile cover.out

##@ Docker

IMG ?= ghcr.io/external-secrets/bitwarden-sdk-server
TAG ?= v0.1.0

docker_image: ## Creates a docker image. Requires `image` and `version` variables on command line
	docker build -t $(IMG):$(VERSION) .

##@ Utilities

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

help:  ## Display this help. Thanks to https://www.thapaliya.com/en/writings/well-documented-makefiles/
ifeq ($(OS),Windows_NT)
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-40s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
else
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-40s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
endif
