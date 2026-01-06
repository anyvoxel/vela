# The old school Makefile, following are required targets. The Makefile is written
# to allow building multiple binaries. You are free to add more targets or change
# existing implementations, as long as the semantics are preserved.
#
#   make              - default to 'build' target
#   make lint         - code analysis
#   make test         - run unit test (or plus integration test)
#   make build        - alias to build-local target
#   make build-local  - build local binary targets
#   make build-linux  - build linux binary targets
#   make container    - build containers
#   $ docker login registry -u username -p xxxxx
#   make push         - push containers
#   make clean        - clean up targets
#
# Not included but recommended targets:
#   make e2e-test
#
# The makefile is also responsible to populate project version information.
#

#
# Tweak the variables based on your project.
#

# This repo's root import path (under GOPATH).
ROOT := github.com/anyvoxel/vela

# Module name.
NAME := vela

# Container image prefix and suffix added to targets.
# The final built images are:
#   $[REGISTRY]/$[IMAGE_PREFIX]$[TARGET]$[IMAGE_SUFFIX]:$[VERSION]
# $[REGISTRY] is an item from $[REGISTRIES], $[TARGET] is an item from $[TARGETS].
IMAGE_PREFIX ?= $(strip )
IMAGE_SUFFIX ?= $(strip )

# Container registries.
REGISTRY ?= 

#
# These variables should not need tweaking.
#

# It's necessary to set this because some environments don't link sh -> bash.
export SHELL := /bin/bash

# It's necessary to set the errexit flags for the bash shell.
export SHELLOPTS := errexit

# Project main package location.
CMD_DIR := ./cmd

# Project output directory.
OUTPUT_DIR := ./bin

# Build directory.
BUILD_DIR := ./build

IMAGE_NAME := $(IMAGE_PREFIX)$(NAME)$(IMAGE_SUFFIX)

# Current version of the project.
VERSION      ?= $(shell git describe --tags --always --dirty)
BRANCH       ?= $(shell git branch | grep \* | cut -d ' ' -f2)
GITCOMMIT    ?= $(shell git rev-parse HEAD)
GITTREESTATE ?= $(if $(shell git status --porcelain),dirty,clean)
BUILDDATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
appVersion   ?= $(VERSION)

# Available cpus for compiling, please refer to https://major.io/2019/04/05/inspecting-openshift-cgroups-from-inside-the-pod/ for more information.
CPUS ?= $(shell /bin/bash hack/read_cpus_available.sh)

# Track code version with Docker Label.
DOCKER_LABELS ?= git-describe="$(shell date -u +v%Y%m%d)-$(shell git describe --tags --always --dirty)"

# Golang standard bin directory.
GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

# Default golang flags used in build and test
# -count: run each test and benchmark 1 times. Set this flag to disable test cache
export GOFLAGS ?= -count=1

#
# Define all targets. At least the following commands are required:
#

# All targets.
.PHONY: lint test build container push

build: build-local

# more info about `GOGC` env: https://github.com/golangci/golangci-lint#memory-usage-of-golangci-lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) --version
	@$(GOLANGCI_LINT) run -v

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BIN_DIR) v2.4.0

test:
	# 1. Now we can use -race flag, the 'nosplit stack overflow' error will not happened at >= go1.20
	#   NOTE: see https://github.com/golang/go/issues/54291 for more detail
	# 2. add -ldflags to prevent 'permission denied' in macos, see https://github.com/agiledragon/gomonkey/issues/70 for more detail.
	@go test -v -race -ldflags="-extldflags="-Wl,-segprot,__TEXT,rwx,rx"" -coverpkg=./... -coverprofile=coverage.out -gcflags="all=-N -l" ./...
	@go tool cover -func coverage.out | tail -n 1 | awk '{ print "Total coverage: " $$3 }'

build-local:
	@go build -v -o $(OUTPUT_DIR)/$(NAME)                                        \
	  -ldflags "-s -w -X $(ROOT)/pkg/utils/version.module=$(NAME)                \
	    -X $(ROOT)/pkg/utils/version.version=$(VERSION)                          \
	    -X $(ROOT)/pkg/utils/version.branch=$(BRANCH)                            \
	    -X $(ROOT)/pkg/utils/version.gitCommit=$(GITCOMMIT)                      \
	    -X $(ROOT)/pkg/utils/version.gitTreeState=$(GITTREESTATE)                \
	    -X $(ROOT)/pkg/utils/version.buildDate=$(BUILDDATE)"                     \
	  $(CMD_DIR);

build-linux:
	/bin/bash -c 'GOOS=linux GOARCH=amd64 GOPATH=/go GOFLAGS="$(GOFLAGS)"        \
	  go build -v -o $(OUTPUT_DIR)/$(NAME)                                       \
	    -ldflags "-s -w -X $(ROOT)/pkg/utils/version.module=$(NAME)              \
	      -X $(ROOT)/pkg/utils/version.version=$(VERSION)                        \
	      -X $(ROOT)/pkg/utils/version.branch=$(BRANCH)                          \
	      -X $(ROOT)/pkg/utils/version.gitCommit=$(GITCOMMIT)                    \
	      -X $(ROOT)/pkg/utils/version.gitTreeState=$(GITTREESTATE)              \
	      -X $(ROOT)/pkg/utils/version.buildDate=$(BUILDDATE)"                   \
		$(CMD_DIR)'

container:
	@docker build -t $(REGISTRY)$(IMAGE_NAME):$(VERSION)                  \
	  --label $(DOCKER_LABELS)                                             \
	  -f $(BUILD_DIR)/Dockerfile .;

push: container
	@docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION);

.PHONY: clean
clean:
	@-rm -vrf ${OUTPUT_DIR} output coverage.out

MOCKGEN := $(BIN_DIR)/mockgen
.PHONY: mock
mock: $(MOCKGEN)
	mockgen -source=pkg/server/server.go -destination=pkg/server/mocks/server.go -package=mocks

$(MOCKGEN):
	go install go.uber.org/mock/mockgen@latest

addheaders:
	@command -v addlicense > /dev/null || go install -v github.com/google/addlicense@v0.0.0-20210428195630-6d92264d7170
	@addlicense -c "The Songlin Yang Authors" -l mit .

PROTOCGO := $(BIN_DIR)/protoc-gen-go
PROTOCGRPC := $(BIN_DIR)/protoc-gen-go-grpc
PROTOCGATEWAY := $(BIN_DIR)/protoc-gen-grpc-gateway
.PHONY: proto
proto: $(PROTOCGO) $(PROTOCGRPC) $(PROTOCGATEWAY)
	@rm -rf ./pb
	@./proto/generate.sh

$(PROTOCGO):
	go install -v google.golang.org/protobuf/cmd/protoc-gen-go@v1.30.0

$(PROTOCGRPC):
	go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

$(PROTOCGATEWAY):
	go install -v github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.15.2
	go install -v github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.15.2
