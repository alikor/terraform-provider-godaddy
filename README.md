# terraform-provider-godaddy

[![CI](https://github.com/alikor/terraform-provider-godaddy/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/alikor/terraform-provider-godaddy/actions/workflows/ci.yml)
[![Release](https://github.com/alikor/terraform-provider-godaddy/actions/workflows/release.yml/badge.svg)](https://github.com/alikor/terraform-provider-godaddy/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/alikor/terraform-provider-godaddy)](https://github.com/alikor/terraform-provider-godaddy/releases)

Terraform provider for managing GoDaddy domains, DNS, and adjacent account-backed domain operations.

## Current implementation status

This repository currently includes:

- provider bootstrap with Terraform Plugin Framework
- GoDaddy client runtime with auth, retries, rate limiting, and typed errors
- normalization helpers for domains, nameservers, and DNS RRsets
- provider data sources:
  - `godaddy_domain`
  - `godaddy_domains`
  - `godaddy_dns_record_set`
  - `godaddy_domain_agreements`
  - `godaddy_domain_actions`
  - `godaddy_domain_forwarding`
  - `godaddy_shopper`
- provider resources:
  - `godaddy_dns_record_set`
  - `godaddy_domain_settings`
  - `godaddy_domain_nameservers`
  - `godaddy_domain_contacts`
  - `godaddy_domain_forwarding`
  - `godaddy_domain_dnssec_records`
- unit tests, provider acceptance tests, Terratest smoke coverage, and mock-backed lifecycle coverage for DNS RRsets

The main remaining work is broader acceptance coverage, more end-to-end Terratests, and docs/examples hardening across the full surface area.

## Documentation

- Provider docs: [docs/index.md](docs/index.md)
- Resource docs: [docs/resources](docs/resources)
- Data source docs: [docs/data-sources](docs/data-sources)
- Examples: [examples](examples)
- Terratest plan: [docs/terratest-plan.md](docs/terratest-plan.md)
- Remaining blockers: [docs/todo.md](docs/todo.md)
- Specification: [docs/specs/godaddy-terraform-provider-spec.md](docs/specs/godaddy-terraform-provider-spec.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Development

```bash
make fmt
make test
make testacc
make testterratest
make build
```

Acceptance tests require real GoDaddy credentials and `TF_ACC=1`.

## Examples

Example provider, data source, and resource configurations live under [examples](examples):

- Provider setup: [examples/provider/main.tf](examples/provider/main.tf)
- Data sources: [examples/data-sources](examples/data-sources)
- Resources: [examples/resources](examples/resources)
