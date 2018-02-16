CONTROLLER_NAME  := ocp-controller
VERSION := $(shell date +%Y%m%d%H%M)
IMAGE := mangirdas/$(CONTROLLER_NAME)
REGISTRY := docker.io

.PHONY: install_deps build build-image publish-image

install_deps:
	glide install

build:
	rm -rf bin/%/$(CONTROLLER_NAME)
	go build -v -i -o bin/$(CONTROLLER_NAME) ./cmd

bin/%/$(CONTROLLER_NAME):
	rm -rf bin/%/$(CONTROLLER_NAME)
	GOOS=$* GOARCH=amd64 go build -v -i -o bin/$*/$(CONTROLLER_NAME) ./cmd

build-image: bin/linux/$(CONTROLLER_NAME)
	docker build . -t $(REGISTRY)/$(IMAGE):$(VERSION)

publish-image:
	docker push $(REGISTRY)/mangirdas/$(CONTROLLER_NAME)

update-codegen:
	./hack/update-codegen.sh