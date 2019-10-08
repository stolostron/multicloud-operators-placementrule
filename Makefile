# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GIT_HOST = github.com/IBM
PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

# Image URL to use all building/pushing image targets
IMG ?= multicloud-operators-placementrule
REGISTRY ?= quay.io/multicloudlab

.PHONY: all
all: check test build images

ifneq ("$(realpath $(DEST))", "$(realpath $(PWD))")
	$(error Please run 'make' from $(DEST). Current directory is $(PWD))
endif

include common/Makefile.common.mk


############################################################
# work section
############################################################
$(GOBIN):
	@echo "create gobin"
	@mkdir -p $(GOBIN)

work: $(GOBIN)

############################################################
# check section
############################################################
check: fmt lint

fmt: format-go format-protos format-python

lint: lint-all

############################################################
# test section
############################################################

test:
	@go test ${TESTARGS} ./...

############################################################
# build section
############################################################

build:
	@common/scripts/gobuild.sh go-repo-template ./cmd

############################################################
# images section
############################################################

images: build build-push-images

build-push-images: config-docker
	@docker build . -f Dockerfile -t $(REGISTRY)/$(IMG):$(VERSION)
	@docker push $(REGISTRY)/$(IMG):$(VERSION)

############################################################
# clean section
############################################################
clean:
	rm -f go-repo-template
