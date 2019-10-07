PROJECT_ROOT := github.com/mayadata-io/storage-provisioner
PKG          := $(PROJECT_ROOT)/pkg
API_GROUPS   := ddp/v1alpha1

PACKAGE_VERSION ?= $(shell git describe --always --tags)
REGISTRY ?= quay.io/mayadata
IMG_NAME ?= dao-storprovisioner

BUILD_LDFLAGS = -X $(PROJECT_ROOT)/build.Hash=$(PACKAGE_VERSION)
GO_FLAGS = -gcflags '-N -l' -ldflags "$(BUILD_LDFLAGS)"

.PHONY: build
build: vendor generated_files unit-test $(IMG_NAME)

$(IMG_NAME):
	@echo "Bulding binary $@ ..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=off \
		go build $(GO_FLAGS) -o $@ cmd/main.go

.PHONY: unit-test
unit-test:
	@echo "Running $@ ..."
	@pkgs="$$(go list ./... | grep -v '/client/generated/')" ; \
		go test $${pkgs}

.PHONY: integration-test
integration-test:
	@echo "Running $@ ..."
	go test -i ./test/integration/...
	PATH="$(PWD)/hack/bin:$(PATH)" go test ./test/integration/... -v -timeout 5m -args -v=6

.PHONY: image
image:
	@echo "Running $@ ..."
	docker build -t $(REGISTRY)/$(IMG_NAME):$(PACKAGE_VERSION) .

.PHONY: push
push: image
	@echo "Running $@ ..."
	@docker push $(REGISTRY)/$(IMG_NAME):$(PACKAGE_VERSION)

.PHONY: vendor
vendor: go.mod go.sum
	@echo "Running $@ ..."
	@GO111MODULE=on go mod download
	@GO111MODULE=on go mod vendor

# I prefer using makefile targets instead of ./hack/update-codegen.sh
# since makefile based targets are more manageable than script based 
# approach. There are some differences between two approaches. 
#
# Makefile uses informer & lister as output package names instead of 
# plural forms i.e. informers & listers.
.PHONY: generated_files
generated_files: vendor deepcopy clientset lister informer

# deepcopy expects client-gen source code to be available at the 
# given vendor location. This source code is installed as a binary 
# i.e. deepcopy-gen at $GOPATH/bin
#
# Finally this installed binary is used to generate deepcopy
.PHONY: deepcopy
deepcopy:
	@GO111MODULE=on go install k8s.io/code-generator/cmd/deepcopy-gen
	@echo "+ Generating deepcopy funcs for $(API_GROUPS)"
	@deepcopy-gen \
		--input-dirs $(PKG)/apis/$(API_GROUPS) \
		--output-file-base zz_generated.deepcopy \
		--go-header-file ./hack/custom-boilerplate.go.txt

# clienset expects client-gen source code to be available at the 
# given vendor location. This source code is installed as a binary 
# i.e. client-gen at $GOPATH/bin
#
# Finally this installed binary is used to generate clienset
.PHONY: clientset
clientset:
	@GO111MODULE=on go install k8s.io/code-generator/cmd/client-gen
	@echo "+ Generating clientset for $(API_GROUPS)"
	@client-gen \
		--fake-clientset=false \
		--input $(API_GROUPS) \
		--input-base $(PKG)/apis \
		--go-header-file ./hack/custom-boilerplate.go.txt \
		--clientset-name versioned \
		--clientset-path $(PROJECT_ROOT)/client/generated/clientset

# lister expects client-gen source code to be available at the 
# given vendor location. This source code is installed as a binary 
# i.e. lister-gen at $GOPATH/bin
#
# Finally this installed binary is used to generate lister
.PHONY: lister
lister:
	@GO111MODULE=on go install k8s.io/code-generator/cmd/lister-gen
	@echo "+ Generating lister for $(API_GROUPS)"
	@lister-gen \
		--input-dirs $(PKG)/apis/$(API_GROUPS) \
		--go-header-file ./hack/custom-boilerplate.go.txt \
		--output-package $(PROJECT_ROOT)/client/generated/lister

# informer expects client-gen source code to be available at the 
# given vendor location. This source code is installed as a binary 
# i.e. informer-gen at $GOPATH/bin
#
# Finally this installed binary is used to generate informer
.PHONY: informer
informer:
	@GO111MODULE=on go install k8s.io/code-generator/cmd/informer-gen
	@echo "+ Generating informer for $(API_GROUPS)"
	@informer-gen \
		--input-dirs $(PKG)/apis/$(API_GROUPS) \
		--output-package $(PROJECT_ROOT)/client/generated/informer \
		--versioned-clientset-package $(PROJECT_ROOT)/client/generated/clientset/versioned \
		--go-header-file ./hack/custom-boilerplate.go.txt \
		--listers-package $(PROJECT_ROOT)/client/generated/lister