IMAGE ?= ghcr.io/nexgencloud/csi-hyperstack/csi
VERSION :=
TAG ?= $(VERSION)

ifeq ($(VERSION),)
  $(error VERSION is not set. Usage: make <target> VERSION=<version> [TAG=<tag>])
endif

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE):$(TAG) --build-arg VERSION=$(VERSION) .

.PHONY: docker-push
docker-push: docker-build
	docker push $(IMAGE):$(TAG)

.PHONY: docker-build-push
docker-build-push:
	docker build -t $(IMAGE):$(TAG) --build-arg VERSION=$(VERSION) --platform=linux/amd64 . --push