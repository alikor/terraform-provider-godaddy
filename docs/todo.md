# GoDaddy Terraform Provider Implementation TODO

This TODO breaks the specification in [docs/specs/godaddy-terraform-provider-spec.md](/Users/madu/code/github.com/alikor/terraform-provider-godaddy/docs/specs/godaddy-terraform-provider-spec.md) into stages with parallel workstreams. The goal is to keep independent tracks moving at the same time while preserving the required dependency order.

## Delivery strategy

- Stage 0 creates the skeleton, shared contracts, and test harness.
- Stage 1 delivers the first usable vertical slice: provider auth, core reads, and DNS RRset support.
- Stage 2 adds existing-domain management surfaces that share the same v1 domain read model.
- Stage 3 adds v2 async surfaces, forwarding, DNSSEC, and action inspection.
- Stage 4 hardens release quality, docs, examples, CI, and manual verification.

## Parallel workstreams

- Track A: repo/bootstrap
  - scaffold provider framework layout
  - Makefile, Go modules, docs generation, release config, CI basics
- Track B: client/runtime
  - auth, endpoint resolution, transport, rate limiting, retries, typed errors, action polling
- Track C: normalization/shared models
  - Terraform models, API models, canonicalization helpers, import ID parsing, validators, plan modifiers
- Track D: read-only surfaces
  - provider schema wiring, shared provider config, data sources
- Track E: managed resources
  - DNS RRset, settings, nameservers, contacts, forwarding, DNSSEC
- Track F: quality
  - unit tests, provider acceptance tests, Terratest suites, examples, generated docs, release verification

## Stage 0: Bootstrap and shared contracts

Goal: get a compilable provider skeleton with the shared building blocks every later stage depends on.

Can run in parallel:

- Track A
  - [ ] Scaffold the repository layout from the Terraform Plugin Framework template.
  - [ ] Add `main.go`, `go.mod`, `GNUmakefile`, `README.md`, `CHANGELOG.md`, `.goreleaser.yml`.
  - [ ] Add `examples/`, `docs/`, `internal/provider/`, `internal/client/`, `internal/normalize/`, `internal/acctest/`, `tools/`.
  - [ ] Add `make build`, `make test`, `make testacc`, `make lint`, `make fmt`, `make docs`, `make install`.
- Track B
  - [ ] Define provider runtime config struct and endpoint resolution rules.
  - [ ] Implement HTTP transport skeleton with auth header injection, optional shopper/app/market headers, and request timeout support.
  - [ ] Implement request logging that redacts secrets when `debug_http=true`.
- Track C
  - [ ] Define shared Terraform models for domain, DNS RRset, contacts, nameservers, forwarding, and DNSSEC.
  - [ ] Implement domain, FQDN, nameserver, DNS RRset, contact, and DNSSEC normalization helpers.
  - [ ] Define import ID parsers and shared diagnostics helpers.
- Track F
  - [ ] Create unit test scaffolding and acceptance test env loading helpers.
  - [ ] Define the Terratest harness and fixture conventions from [docs/terratest-plan.md](/Users/madu/code/github.com/alikor/terraform-provider-godaddy/docs/terratest-plan.md).

Exit criteria:

- [ ] `go test ./...` runs against the scaffold.
- [ ] `terraform providers schema -json` works for the provider shell.
- [ ] docs generation pipeline is wired, even if docs are still minimal.

## Stage 1: Core provider, reads, and DNS RRsets

Goal: deliver the first production-capable slice centered on DNS and low-risk reads.

Dependencies:

- Requires Stage 0 shared config, client skeleton, normalization helpers, and test harness.

Can run in parallel:

- Track B
  - [ ] Implement rate limiter keyed by normalized endpoint template.
  - [ ] Implement retry policy for `429`, transient `5xx`, and transport errors.
  - [ ] Implement typed error parsing and Terraform diagnostic conversion.
  - [ ] Implement v1 client methods for domains list/detail, agreements, and DNS record set CRUD.
- Track D
  - [ ] Implement provider schema with env var fallbacks and config validation.
  - [ ] Implement `godaddy_domain` data source basic v1 read path.
  - [ ] Implement `godaddy_domains` data source.
  - [ ] Implement `godaddy_dns_record_set` data source.
- Track E
  - [ ] Implement `godaddy_dns_record_set` resource.
  - [ ] Enforce RRset ownership rules and import-required create behavior.
  - [ ] Implement replace behavior for `domain`, `type`, and `name`.
- Track C
  - [ ] Finalize DNS RRset validators for supported types and nested field rules.
  - [ ] Finalize record ordering and canonical state representation.
- Track F
  - [ ] Add unit tests for endpoint resolution, auth headers, retry logic, error parsing, DNS normalization, and import ID parsing.
  - [ ] Add acceptance tests for `godaddy_dns_record_set`.
  - [ ] Add smoke Terratests for provider auth, domain read, and DNS RRset lifecycle.

Exit criteria:

- [ ] `godaddy_domain`, `godaddy_domains`, and `godaddy_dns_record_set` data source are working.
- [ ] `godaddy_dns_record_set` create, import, update, delete, and replace behavior are idempotent.
- [ ] No perpetual diffs for supported RRset types.

## Stage 2: Existing-domain management surfaces

Goal: add adopt-and-reconcile resources for existing domains using mostly v1 endpoints, plus the shared action poller needed by nameserver and contact v2 paths.

Dependencies:

- Requires Stage 1 provider wiring and v1 domain read models.

Can run in parallel:

- Track B
  - [ ] Implement `ResolveCustomerID` with runtime cache and shopper lookup fallback.
  - [ ] Implement shopper v1 client methods.
  - [ ] Implement reusable v2 action poller and action list/get client methods.
- Track D
  - [ ] Implement `godaddy_shopper` data source.
  - [ ] Implement `godaddy_domain_agreements` data source.
- Track E1: domain settings
  - [ ] Implement `godaddy_domain_settings` resource.
  - [ ] Add consent-aware `ModifyPlan` validation for exposure flags.
  - [ ] Implement state-only unmanage delete semantics.
- Track E2: nameservers
  - [ ] Implement `godaddy_domain_nameservers` resource.
  - [ ] Support preferred v2 update path with v1 fallback when `customer_id` is unavailable.
  - [ ] Add nameserver eligibility and 2FA-related diagnostics.
- Track E3: contacts
  - [ ] Implement `godaddy_domain_contacts` resource.
  - [ ] Support v1 default path and v2 path when `identity_document_id` is set.
- Track C
  - [ ] Finalize contact mappers and nameserver normalization.
  - [ ] Finalize shared plan modifiers and validators for state-only resources.
- Track F
  - [ ] Add unit tests for customer ID resolution, action polling, contact mapping, and nameserver normalization.
  - [ ] Add acceptance tests for `godaddy_domain_settings`, `godaddy_domain_nameservers`, `godaddy_domain_contacts`, `godaddy_domain_agreements`, and `godaddy_shopper`.
  - [ ] Add Terratests covering import/no-op flows and state-only destroy semantics.

Exit criteria:

- [ ] All three existing-domain management resources can import and reconcile against a pre-existing domain.
- [ ] Consent validation, v2 polling, and state-only deletes behave as documented.

## Stage 3: Forwarding, DNSSEC, and advanced v2 reads

Goal: finish the v1 scope with async v2 resources and advanced inspection surfaces.

Dependencies:

- Requires Stage 2 customer ID resolution and reusable action poller.

Can run in parallel:

- Track B
  - [ ] Implement forwarding v2 client methods.
  - [ ] Implement DNSSEC v2 client methods.
  - [ ] Implement v2 domain detail reads with include support and partial-success handling.
- Track D
  - [ ] Extend `godaddy_domain` data source to support advanced includes, partial responses, and sensitive `auth_code`.
  - [ ] Implement `godaddy_domain_actions` data source.
  - [ ] Implement `godaddy_domain_forwarding` data source.
- Track E1: forwarding
  - [ ] Implement `godaddy_domain_forwarding` resource with import-required create.
  - [ ] Support masked, temporary, and permanent redirect variants.
- Track E2: DNSSEC
  - [ ] Implement `godaddy_domain_dnssec_records` resource as full-set ownership.
  - [ ] Implement add-before-remove diff reconciliation.
  - [ ] Error when v2 partial success omits `dnssecRecords`.
- Track C
  - [ ] Finalize DNSSEC canonical ordering and diff logic.
  - [ ] Finalize forwarding schema validators and mapping.
- Track F
  - [ ] Add unit tests for DNSSEC diff calculation and v2 partial-success handling.
  - [ ] Add acceptance tests for forwarding, domain actions, advanced domain reads, and DNSSEC lifecycle.
  - [ ] Add Terratests for forwarding CRUD, DNSSEC rollover, and async polling timeout behavior.

Exit criteria:

- [ ] Forwarding CRUD is stable.
- [ ] DNSSEC add/remove reconciliation is stable and safe for rollover.
- [ ] Advanced v2 reads expose partial data correctly.

## Stage 4: Release hardening

Goal: make the provider shippable.

Can run in parallel:

- Track A
  - [ ] Finalize `.goreleaser.yml` for linux, darwin, and windows on amd64 and arm64.
  - [ ] Wire registry packaging and version injection.
- Track D
  - [ ] Finish `MarkdownDescription` coverage on every provider, resource, and data source field.
  - [ ] Add required examples under `examples/provider`, `examples/resources`, and `examples/data-sources`.
- Track F
  - [ ] Wire `tfplugindocs` into `make docs`.
  - [ ] Add CI jobs for format, lint, unit tests, docs generation, and gated acceptance/Terratest runs.
  - [ ] Run the manual verification matrix on OTE and Production-eligible accounts.
  - [ ] Verify import for every resource and confirm drift-free second apply behavior.

Exit criteria:

- [ ] Generated docs are complete and accurate.
- [ ] Release packaging works locally and in CI.
- [ ] Manual verification passes for auth, DNS, settings, nameservers, forwarding, and DNSSEC where supported.

## Cross-cutting backlog

- [ ] Keep all resources on explicit computed `id` attributes.
- [ ] Ensure every mutating resource performs read-after-write canonical refresh.
- [ ] Keep resource ownership boundaries strict and non-overlapping.
- [ ] Ensure secrets never appear in logs, state, or test output where forbidden.
- [ ] Serialize acceptance and Terratest cases that mutate the same domain.

## Suggested team split

- Engineer 1
  - Track A and release tooling
- Engineer 2
  - Track B client/runtime
- Engineer 3
  - Track C normalization/shared models
- Engineer 4
  - Track D data sources/provider wiring
- Engineer 5
  - Track E resources
- Engineer 6
  - Track F tests/docs/examples

If the team is smaller, combine Tracks A and F, and combine Tracks C and D.
