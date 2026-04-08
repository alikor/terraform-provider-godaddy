GO ?= go
TERRAFORM ?= terraform
DOCKER ?= docker
VERSION ?= dev

.PHONY: build test testacc testterratest teste2e lint fmt docs install docker-fmt docker-test docker-build docker-smoke docker-ci

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

install:
	$(GO) install -ldflags="-X main.version=$(VERSION)" .

docker-fmt:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target fmt-check .

docker-test:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target test .

docker-build:
	DOCKER_BUILDKIT=1 $(DOCKER) build --target build --build-arg VERSION=$(VERSION) .

docker-smoke:
	DOCKER_BUILDKIT=1 $(DOCKER) build \
		--target terratest-smoke \
		--build-arg GODADDY_ENDPOINT=$${GODADDY_ENDPOINT:-ote} \
		--secret id=godaddy_api_key,env=GODADDY_API_KEY \
		--secret id=godaddy_api_secret,env=GODADDY_API_SECRET \
		--secret id=godaddy_test_domain,env=GODADDY_TEST_DOMAIN \
		.

docker-ci: docker-fmt docker-test docker-build
