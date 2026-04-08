GO ?= go
TERRAFORM ?= terraform
VERSION ?= dev

.PHONY: build test testacc testterratest teste2e lint fmt docs install

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
