# syntax=docker/dockerfile:1.7

FROM golang:1.26-bookworm AS toolchain

WORKDIR /workspace

COPY --from=hashicorp/terraform:1.5.7 /bin/terraform /usr/local/bin/terraform

ENV CGO_ENABLED=0 \
    GOFLAGS=-buildvcs=false

FROM toolchain AS deps

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM deps AS source

COPY . .

FROM source AS fmt-check

RUN test -z "$(gofmt -l .)"

FROM source AS test

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test ./...

FROM source AS docs-check

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    rm -rf /tmp/docs-check && \
    mkdir -p /tmp/docs-check && \
    cp -R docs/index.md docs/data-sources docs/resources /tmp/docs-check/ && \
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name godaddy && \
    diff -ru /tmp/docs-check/index.md docs/index.md && \
    diff -ru /tmp/docs-check/data-sources docs/data-sources && \
    diff -ru /tmp/docs-check/resources docs/resources

FROM source AS build

ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    go build -ldflags="-X main.version=${VERSION}" -o /out/terraform-provider-godaddy .

FROM scratch AS artifact

COPY --from=build /out/terraform-provider-godaddy /terraform-provider-godaddy

FROM source AS terratest-smoke

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test ./test/terratest/... -run 'TestDNSRecordSetLifecycleWithMockAPI' -v -count=1
