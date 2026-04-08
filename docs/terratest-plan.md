# Terratest Plan for terraform-provider-godaddy

This plan defines how Terratest should complement unit tests and Terraform Plugin Framework acceptance tests.

## Purpose

- Use unit tests for pure Go logic.
- Use Terraform provider acceptance tests for resource and data source implementation details.
- Use Terratest for black-box, end-to-end verification against the built provider binary and real Terraform workflows.

Terratest is most valuable here for:

- provider installation and smoke validation
- example configuration verification
- import and second-apply idempotence checks
- high-risk domain workflows that are easiest to assert from the Terraform CLI boundary

## Recommended test layout

```text
test/
└── terratest/
    ├── helpers/
    │   ├── provider.go
    │   ├── env.go
    │   ├── random.go
    │   └── cleanup.go
    ├── fixtures/
    │   ├── provider_smoke/
    │   ├── dns_record_set/
    │   ├── domain_settings/
    │   ├── domain_nameservers/
    │   ├── domain_contacts/
    │   ├── domain_forwarding/
    │   ├── domain_dnssec_records/
    │   └── data_sources/
    ├── provider_smoke_test.go
    ├── dns_record_set_test.go
    ├── domain_settings_test.go
    ├── domain_nameservers_test.go
    ├── domain_contacts_test.go
    ├── domain_forwarding_test.go
    ├── domain_dnssec_records_test.go
    └── data_sources_test.go
```

## Environment contract

Terratests should use the same env vars as provider acceptance tests:

- `TF_ACC=1`
- `GODADDY_API_KEY`
- `GODADDY_API_SECRET`
- `GODADDY_ENDPOINT`
- `GODADDY_SHOPPER_ID`
- `GODADDY_CUSTOMER_ID`
- `GODADDY_APP_KEY`
- `GODADDY_TEST_DOMAIN`
- `GODADDY_TEST_FORWARD_FQDN`
- `GODADDY_TEST_FORWARD_URL`

Additional Terratest-specific env vars:

- `GODADDY_TEST_NS1`
- `GODADDY_TEST_NS2`
- `GODADDY_TEST_IDENTITY_DOCUMENT_ID`
- `GODADDY_TEST_KEEP_RESOURCES`

## Shared Terratest helpers

- `helpers/provider.go`
  - build the provider binary or reuse a locally built binary
  - write a temporary CLI config that uses the local provider mirror
- `helpers/env.go`
  - load required env vars and skip cleanly when prerequisites are missing
- `helpers/random.go`
  - generate disposable subdomain prefixes and unique Terraform workspace names
- `helpers/cleanup.go`
  - centralize safe best-effort cleanup for forwarding, DNS, and DNSSEC

## Suite definitions

### 1. Provider smoke suite

Goal:

- prove Terraform can initialize the local provider and read current account/domain context

Checks:

- `terraform init`
- `terraform validate`
- `terraform plan`
- read `godaddy_domain` for `GODADDY_TEST_DOMAIN`
- optional read `godaddy_shopper` when `GODADDY_SHOPPER_ID` is present

### 2. DNS RRset suite

Goal:

- verify the first full lifecycle slice using disposable hostnames

Checks:

- apply `A`, `TXT`, `MX`, and `SRV` RRsets
- run import verification
- run a second `terraform apply` and assert no changes
- destroy and confirm the RRset is gone
- run an already-exists case and assert the plan/apply fails with import guidance

Parallelism:

- safe to run parallel by unique hostname
- serialize per `(domain, type, name)` tuple

### 3. Domain settings suite

Goal:

- verify adopt-and-reconcile behavior on an existing domain

Checks:

- import existing settings resource
- toggle `locked`
- toggle `renew_auto`
- verify consent-required plan failure when enabling WHOIS exposure without consent
- destroy state only and confirm the domain remains unchanged remotely

Parallelism:

- serialize per domain

### 4. Nameservers suite

Goal:

- verify nameserver replacement and async v2 behavior

Checks:

- import existing nameserver state
- apply alternative nameserver set
- verify poll completion and final state
- verify second apply is a no-op
- destroy state only

Safety:

- run only in a dedicated suite and only when explicit test nameserver env vars are present
- never run in parallel with any other test mutating the same domain

### 5. Contacts suite

Goal:

- verify full contact-set ownership

Checks:

- import current contacts
- update all four contact roles via v1 path
- optionally run v2 identity document flow when `GODADDY_TEST_IDENTITY_DOCUMENT_ID` is present
- verify second apply is a no-op
- destroy state only

Parallelism:

- serialize per domain

### 6. Forwarding suite

Goal:

- verify forwarding CRUD through the Terraform CLI boundary

Checks:

- create forwarding for `GODADDY_TEST_FORWARD_FQDN`
- import the created object
- update redirect type and URL
- verify masked forwarding path
- destroy and confirm remote deletion
- verify existing-forward conflict returns import guidance

Parallelism:

- safe only if each test uses a unique FQDN

### 7. DNSSEC suite

Goal:

- verify full-set ownership and rollover-safe reconciliation

Checks:

- create DNSSEC set
- import DNSSEC resource
- update with add-before-remove rollover scenario
- verify empty remote set removes resource from state
- destroy and confirm removal
- verify timeout/error propagation from async action polling

Safety:

- run only on a domain/TLD known to support DNSSEC testing
- serialize completely

### 8. Data sources suite

Goal:

- verify read-only surfaces and example usability

Checks:

- `godaddy_domain`
- `godaddy_domains`
- `godaddy_dns_record_set`
- `godaddy_domain_agreements`
- `godaddy_domain_actions`
- `godaddy_domain_forwarding`
- `godaddy_shopper`

## Execution model

- `make testacc` should continue to run Terraform Plugin Framework acceptance tests.
- Add `make testterratest` for Terratest suites.
- Add `make teste2e` as a convenience target that runs both acceptance tests and Terratest.

Recommended commands:

```make
testterratest:
	go test ./test/terratest/... -v -timeout 90m

teste2e: testacc testterratest
```

## CI strategy

- PR CI
  - run unit tests, lint, docs generation, and optional smoke Terratest against a mocked or explicitly enabled environment
- gated pre-release CI
  - run full acceptance tests and Terratest against OTE or a dedicated production-safe account
- release verification
  - run provider smoke, DNS RRset, and example suites against the built release artifact

## Exit criteria

- every required v1 resource has at least one Terratest lifecycle suite
- every critical example can pass `terraform init`, `plan`, and `apply` where appropriate
- second apply is confirmed to be a no-op for all managed resources
- destructive tests are isolated, serialized, and use dedicated test inputs
