# GoDaddy Terraform Provider Remaining Work

This file tracks the real blockers between the current `0.2.x` state and a credible `1.0.0` release. Earlier stage planning has been superseded by the implemented provider surface now present in the repository.

## Current baseline

Implemented today:

- provider runtime, auth, retry, rate limiting, typed errors, and action polling
- data sources:
  - `godaddy_domain`
  - `godaddy_domains`
  - `godaddy_dns_record_set`
  - `godaddy_domain_agreements`
  - `godaddy_domain_actions`
  - `godaddy_domain_forwarding`
  - `godaddy_shopper`
- resources:
  - `godaddy_dns_record_set`
  - `godaddy_domain_settings`
  - `godaddy_domain_nameservers`
  - `godaddy_domain_contacts`
  - `godaddy_domain_forwarding`
  - `godaddy_domain_dnssec_records`
- unit tests
- provider acceptance harness
- Terratest smoke coverage
- generated provider docs
- Docker-first CI and release packaging

## Blockers for 1.0.0

### 1. Live acceptance coverage

The main remaining blocker is real-account verification with an OTE or production-eligible account that exposes at least one API-visible domain.

Still required:

- acceptance tests for:
  - `godaddy_domain_settings`
  - `godaddy_domain_nameservers`
  - `godaddy_domain_contacts`
  - `godaddy_domain_forwarding`
  - `godaddy_domain_dnssec_records`
  - `godaddy_domain_agreements`
  - `godaddy_shopper`
  - advanced `godaddy_domain` include paths
- import verification for every managed resource
- drift-free second apply verification for every managed resource

Current status:

- the acceptance harness works and skips cleanly when the configured account has no accessible domains
- current OTE credentials tested so far return either empty `/v1/domains` results or `domain not found for shopper`

### 2. Broader Terratest coverage

Still required:

- import/no-op Terratests for managed resources
- state-only destroy coverage for:
  - `godaddy_domain_settings`
  - `godaddy_domain_nameservers`
  - `godaddy_domain_contacts`
- forwarding CRUD coverage
- DNSSEC rollover and add-before-remove coverage
- async polling timeout/error path coverage

Current status:

- smoke Terratests exist for:
  - provider prereqs
  - domain read planning
  - DNS record set planning

### 3. Resource behavior hardening

Still required:

- stronger plan-time validation for the remaining state-only resources, especially around unmanage semantics
- further verification of nameserver eligibility and 2FA-related error handling against live domains
- further live verification of partial-response handling for v2 advanced reads and DNSSEC

Current status:

- plan-time validation now exists for:
  - `godaddy_domain_settings` consent transitions
  - `godaddy_domain_nameservers` minimum nameserver set enforcement
- runtime behavior exists for the remaining paths
- live validation is still limited by account/domain availability

### 4. Documentation polish

Still required:

- review generated docs for narrative examples and caveats, not just schema completeness
- add troubleshooting notes for:
  - shopper ID vs customer ID
  - OTE account/domain visibility
  - Terraform Registry signing key setup

Current status:

- `tfplugindocs` is wired
- generated docs are checked into `docs/`
- CI now verifies docs stay in sync

## Recommended next milestone

The highest-value next milestone is:

1. obtain one OTE account with an API-visible domain
2. run and expand acceptance coverage for all existing-domain resources
3. add the remaining Terratest lifecycle/import coverage
4. cut `1.0.0` only after those live verifications pass
