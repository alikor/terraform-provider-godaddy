# terraform-provider-godaddy

Terraform provider for managing GoDaddy domains, DNS, and adjacent account-backed domain operations.

## Current implementation status

This repository currently includes:

- provider bootstrap with Terraform Plugin Framework
- GoDaddy client runtime with auth, retries, rate limiting, and typed errors
- normalization helpers for domains, nameservers, and DNS RRsets
- Stage 1 provider surfaces:
  - `godaddy_domain`
  - `godaddy_domains`
  - `godaddy_dns_record_set` data source
  - `godaddy_domain_agreements`
  - `godaddy_shopper`
  - `godaddy_dns_record_set` resource
- unit tests and Terratest scaffolding

Later stage resources from the spec remain to be implemented.

## Development

```bash
make fmt
make test
make build
```

Acceptance tests require real GoDaddy credentials and `TF_ACC=1`.
