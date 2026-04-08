GO ?= go
TERRAFORM ?= terraform
DOCKER ?= docker
VERSION ?= $(shell ./tools/version.sh)
DIST_DIR ?= dist/docker

.PHONY: build test testacc testterratest teste2e lint fmt docs docs-check install version docker-fmt docker-test docker-docs-check docker-build docker-artifact docker-smoke docker-ci

version:
	@printf '%s\n' "$(VERSION)"

build:
	$(GO) build -ldflags="-X main.version=$(VERSION)" ./...

test:
	$(GO) test ./...

testacc:
	TF_ACC=1 $(GO) test ./internal/provider/... -v -count=1

testterratest:
	$(GO) test ./test/terratest/... -v -count=1 -timeout 90m

teste2e: testacc testterratest

lint:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

docs:
	$(GO) run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name godaddy

docs-check:
	rm -rf .tmp-docs-check
	mkdir -p .tmp-docs-check
	cp -R docs/index.md docs/data-sources docs/resources .tmp-docs-check/
	$(GO) run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name godaddy
	diff -ru .tmp-docs-check/index.md docs/index.md
	diff -ru .tmp-docs-check/data-sources docs/data-sources
	diff -ru .tmp-docs-check/resources docs/resources
	rm -rf .tmp-docs-check

install:
	$(GO) install -ldflags="-X main.version=$(VERSION)" .

docker-fmt:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target fmt-check .

docker-test:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target test .

docker-docs-check:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target docs-check .

docker-build:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target build --build-arg VERSION=$(VERSION) .

docker-artifact:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	DOCKER_BUILDKIT=1 $(DOCKER) build --target artifact --build-arg VERSION=$(VERSION) --output type=local,dest=$(DIST_DIR) .

docker-smoke:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target terratest-smoke .

docker-ci: docker-fmt docker-test docker-docs-check docker-build
