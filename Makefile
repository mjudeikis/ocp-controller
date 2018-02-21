CONTROLLER_NAME  := tagcontroller
#VERSION := $(shell date +%Y%m%d%H%M)
VERSION := latest
IMAGE := mangirdas/$(CONTROLLER_NAME)
REGISTRY := docker.io
ARCH := linux 

.PHONY: install_deps build build-image publish-image

install_deps:
	glide install

build:
	rm -rf bin/%/$(CONTROLLER_NAME)
	GOOS=$(ARCH) GOARCH=amd64 go build -v -i -o bin/$(CONTROLLER_NAME) ./cmd

bin/%/$(CONTROLLER_NAME):
	rm -rf bin/%/$(CONTROLLER_NAME)
	GOOS=$* GOARCH=amd64 go build -v -i -o bin/$*/$(CONTROLLER_NAME) ./cmd

build-image: build
	docker build . -t $(REGISTRY)/$(IMAGE):$(VERSION)

publish-image: build-image
	docker push $(REGISTRY)/mangirdas/$(CONTROLLER_NAME):$(VERSION)

update-codegen:
	./hack/update-codegen.sh