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

FROM source AS build

ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    go build -ldflags="-X main.version=${VERSION}" -o /out/terraform-provider-godaddy .

FROM scratch AS artifact

COPY --from=build /out/terraform-provider-godaddy /terraform-provider-godaddy

FROM source AS terratest-smoke

ARG GODADDY_ENDPOINT=ote

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=secret,id=godaddy_api_key \
    --mount=type=secret,id=godaddy_api_secret \
    --mount=type=secret,id=godaddy_test_domain \
    GODADDY_API_KEY="$(cat /run/secrets/godaddy_api_key)" && \
    GODADDY_API_SECRET="$(cat /run/secrets/godaddy_api_secret)" && \
    GODADDY_TEST_DOMAIN="$(cat /run/secrets/godaddy_test_domain)" && \
    export TF_ACC=1 && \
    export GODADDY_ENDPOINT="${GODADDY_ENDPOINT}" && \
    export GODADDY_API_KEY GODADDY_API_SECRET GODADDY_TEST_DOMAIN && \
    export TF_VAR_godaddy_api_key="${GODADDY_API_KEY}" && \
    export TF_VAR_godaddy_api_secret="${GODADDY_API_SECRET}" && \
    export TF_VAR_godaddy_endpoint="${GODADDY_ENDPOINT}" && \
    export TF_VAR_domain="${GODADDY_TEST_DOMAIN}" && \
    go test ./test/terratest/... -run TestDNSRecordSetPlan -v -count=1
