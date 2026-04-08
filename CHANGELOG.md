# Changelog

## Unreleased

## 0.3.0 - 2026-04-08

- Added plan-time validation for `godaddy_domain_settings` consent-gated WHOIS exposure transitions.
- Added plan-time validation for `godaddy_domain_nameservers` minimum nameserver requirements.
- Added mock-backed DNS RRset lifecycle Terratest coverage for apply, import, no-op plan, and destroy flows.
- Fixed DNS RRset state normalization so unset optional numeric fields remain `null` instead of drifting to `0`.
- Switched CI smoke coverage from live OTE domain reads to deterministic local mock lifecycle testing.
- Added GitHub Actions and latest release badges to the root README.

- Added v2 nameserver updates with async action polling and v1 fallback.
- Added provider acceptance tests for `godaddy_domain` and `godaddy_dns_record_set`.
- Expanded Terratest coverage with a domain read smoke fixture.
- Added examples for domains, shopper, nameservers, settings, forwarding, contacts, and DNSSEC resources.
