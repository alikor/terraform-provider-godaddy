# GoDaddy Terraform Provider — Detailed Build Specification

**Document status:** Normative build specification  
**Audience:** AI coding agent or human engineer implementing the provider from scratch  
**Snapshot date:** 2026-04-08  
**Primary objective:** Build a Terraform provider that automates a GoDaddy account, with priority on domain DNS and adjacent existing-domain management.

---

## 1. Executive summary

Build a new Terraform provider named `godaddy` in Go using the **Terraform Plugin Framework**. The first production-capable version should focus on **existing domain management**, especially:

- DNS RRsets
- domain settings (lock, auto-renew, WHOIS exposure flags)
- nameservers
- contacts
- forwarding
- DNSSEC
- read-only data sources for domains, DNS, agreements, shopper/account context, and action status

Do **not** make domain purchase / renewal / transfer / privacy purchase / subscription cancellation ordinary CRUD resources in the first release. Those operations are billing- and consent-affecting, often asynchronous, and map better to **Terraform actions** (later phase) than to long-lived managed resources.

This provider must be opinionated where Terraform and the GoDaddy API do not align perfectly. In particular:

1. Manage DNS as **RRsets**, not single records.
2. Split mutable domain concerns into **non-overlapping resources**.
3. Treat most domain-level resources as **adopt-and-reconcile** wrappers over an already-existing domain.
4. Use **v1** endpoints for the simplest stable CRUD where possible, and **v2** endpoints when required for async actions, forwarding, and DNSSEC.
5. Implement a robust **rate limiter**, **retry policy**, and **async action poller**.

Normative sources are listed in Section 3.

---

## 2. Scope, goals, and non-goals

### 2.1 Must-have goals

The initial provider MUST support:

- authenticating to GoDaddy OTE and Production
- self-serve accounts
- optional reseller/subaccount context via `X-Shopper-Id`
- management of DNS RRsets for domains using GoDaddy DNS
- management of an existing domain's:
  - lock state
  - auto-renew state
  - WHOIS exposure flags
  - nameservers
  - contacts
  - URL forwarding
  - DNSSEC records
- import of existing remote objects/settings into Terraform state
- data sources for domain inspection and discovery
- acceptance tests, generated docs, examples, and release packaging

### 2.2 Explicit non-goals for v1

The initial provider MUST NOT implement these as normal CRUD resources:

- domain registration/purchase
- domain renewal
- transfer-in / transfer-out workflows
- privacy purchase / delete
- redemption / trade flows
- subscription cancellation as managed resource lifecycle
- shopper deletion
- any operation that irreversibly purchases or cancels a paid product as part of a normal `terraform apply` for a CRUD resource

These can be added later as **Terraform actions** or explicit opt-in surfaces.

### 2.3 Product constraints that must shape the design

The design MUST account for the following documented GoDaddy constraints:

- GoDaddy uses `Authorization: sso-key <api_key>:<api_secret>` for API auth.
- OTE and Production use different base URLs and different keys.
- Each GoDaddy API endpoint is limited to **60 requests per minute**.
- Some production Domains functionality is gated by account eligibility.
- Purchases require a **Good as Gold** account.
- Some protected/high-value nameserver updates may require 2FA not supported through the API.
- `shopperId` and `customerId` are different identifiers.
- Some v2 domain operations are asynchronous and expose action status objects.

---

## 3. Normative source list

Use these as the primary source of truth during implementation.

### GoDaddy

- **[GD-GETSTARTED]** GoDaddy API Get Started  
  `https://developer.godaddy.com/getstarted`

- **[GD-DOMAINS-DOC]** Domains API docs landing page  
  `https://developer.godaddy.com/doc/endpoint/domains`

- **[GD-DOMAINS-SWAGGER]** Domains Swagger (OpenAPI 2.0 / Swagger 2.0)  
  `https://developer.godaddy.com/swagger/swagger_domains.json`

- **[GD-SHOPPERS-DOC]** Shoppers API docs landing page  
  `https://developer.godaddy.com/doc/endpoint/shoppers`

- **[GD-SHOPPERS-SWAGGER]** Shoppers Swagger  
  `https://developer.godaddy.com/swagger/swagger_shoppers.json`

- **[GD-SUBSCRIPTIONS-DOC]** Subscriptions API docs landing page  
  `https://developer.godaddy.com/doc/endpoint/subscriptions`

- **[GD-SUBSCRIPTIONS-SWAGGER]** Subscriptions Swagger  
  `https://developer.godaddy.com/swagger/swagger_subscriptions.json`

### HashiCorp / Terraform

- **[HC-FRAMEWORK]** Terraform Plugin Framework overview  
  `https://developer.hashicorp.com/terraform/plugin/framework`

- **[HC-RESOURCES]** Framework resources  
  `https://developer.hashicorp.com/terraform/plugin/framework/resources`

- **[HC-IMPORT]** Resource import  
  `https://developer.hashicorp.com/terraform/plugin/framework/resources/import`

- **[HC-ACTIONS-IMPL]** Implement actions  
  `https://developer.hashicorp.com/terraform/plugin/framework/actions/implementation`

- **[HC-ACTIONS-INVOKE]** Terraform actions language usage  
  `https://developer.hashicorp.com/terraform/language/invoke-actions`

- **[HC-ACCTEST]** Acceptance tests  
  `https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests`

- **[HC-DOCS]** Terraform Registry provider docs generation/publication  
  `https://developer.hashicorp.com/terraform/registry/providers/docs`

- **[HC-WRITEONLY]** Write-only arguments  
  `https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-arguments`

- **[HC-IDENTITY]** Resource identity  
  `https://developer.hashicorp.com/terraform/plugin/framework/resources/identity`

- **[HC-SCAFFOLD]** Terraform provider scaffolding framework repository  
  `https://github.com/hashicorp/terraform-provider-scaffolding-framework`

---

## 4. High-level design decisions

### 4.1 Use the Terraform Plugin Framework

The provider MUST be implemented with `terraform-plugin-framework`, not SDKv2. This is HashiCorp's recommended path for new providers. Resource, data source, and later action implementations should all use the framework patterns from [HC-FRAMEWORK].

### 4.2 Model DNS as RRsets, not individual records

GoDaddy's DNS endpoints operate naturally on **sets of records** for a `(domain, type, name)` tuple and also allow replacing all records of a type or the entire zone. Because Terraform resources assume a single resource owns a single remote object, a resource such as:

- `godaddy_dns_record_set` => one RRset

is the correct shape.

A hypothetical `godaddy_dns_record` resource is **forbidden** because two Terraform resources that manage different records inside the same RRset would overwrite each other via the GoDaddy API.

### 4.3 Split domain concerns into separate field-owning resources

The provider MUST avoid field overlap between resources. Use the following ownership map:

- `godaddy_domain_settings`
  - owns: `locked`, `renew_auto`, `expose_registrant_organization`, `expose_whois`
- `godaddy_domain_nameservers`
  - owns: `name_servers`
- `godaddy_domain_contacts`
  - owns: registrant/admin/tech/billing contacts
- `godaddy_dns_record_set`
  - owns: one DNS RRset
- `godaddy_domain_forwarding`
  - owns: one forwarding rule for one FQDN
- `godaddy_domain_dnssec_records`
  - owns: the entire DNSSEC record set for a domain

This decomposition is intentional and MUST be preserved.

### 4.4 Prefer v1 when it is simpler and stable; use v2 where necessary

Use the following rule:

- **Use v1** for:
  - domain detail/settings
  - domain contacts (default path)
  - DNS records
  - domain list
  - agreements
- **Use v2** for:
  - forwarding
  - DNSSEC
  - domain nameserver updates when `customer_id` is available
  - domain action polling
  - advanced domain data source reads that need `actions`, `dnssecRecords`, `registryStatusCodes`, or v2 contact structures
  - contact updates only when `identity_document_id` is used

### 4.5 Do not codegen the entire GoDaddy API client

The implementation SHOULD use a **hand-written thin client** for the endpoints actually needed. Do not generate the full Swagger into the provider.

Reason:

- the GoDaddy spec is broad and includes many surfaces this provider will not initially manage
- a hand-written client is simpler to control for retries, partial-success behavior, normalization, and Terraform-specific error handling
- endpoint coverage is limited enough to keep a bespoke client maintainable

### 4.6 Use explicit `id` attributes in all resources

The framework does not implicitly add a root `id` attribute. Every managed resource in this provider MUST define an explicit computed `id` attribute.

Resource identity (Terraform 1.12+) is optional later. MVP SHOULD use `id` only.

---

## 5. Provider UX

### 5.1 Provider name and type name

Provider type name MUST be `godaddy`.

All resource and data source names MUST be prefixed with `godaddy_`.

Examples:

- `godaddy_dns_record_set`
- `godaddy_domain_settings`
- `godaddy_domain_nameservers`
- `godaddy_domain_contacts`
- `godaddy_domain_forwarding`
- `godaddy_domain_dnssec_records`
- `data "godaddy_domain" ...`

### 5.2 Provider configuration schema

Implement the following provider arguments:

#### Required or strongly expected

- `api_key` — string, sensitive
- `api_secret` — string, sensitive

#### Optional

- `endpoint` — string enum: `production` or `ote`, default `production`
- `base_url` — string, optional override for tests; if set, overrides `endpoint`
- `shopper_id` — string, optional
- `customer_id` — string, optional
- `app_key` — string, sensitive, optional; required only for subscriptions surfaces
- `market_id` — string, optional, default `en-US`
- `request_timeout` — duration string or integer seconds, default 30s
- `poll_interval` — duration string or integer seconds, default 5s
- `max_retries` — integer, default 5
- `rate_limit_rpm` — integer, default 50, max 60
- `user_agent_suffix` — string, optional
- `debug_http` — bool, default false

#### Environment variable fallbacks

The provider SHOULD support these environment variables:

- `GODADDY_API_KEY`
- `GODADDY_API_SECRET`
- `GODADDY_ENDPOINT`
- `GODADDY_BASE_URL`
- `GODADDY_SHOPPER_ID`
- `GODADDY_CUSTOMER_ID`
- `GODADDY_APP_KEY`
- `GODADDY_MARKET_ID`

### 5.3 Provider behavior rules

1. If `base_url` is set, use it exactly after trimming a trailing slash.
2. Else resolve:
   - `ote` => `https://api.ote-godaddy.com`
   - `production` => `https://api.godaddy.com`
3. Always send:
   - `Authorization: sso-key <api_key>:<api_secret>`
4. Send `X-Shopper-Id` only if `shopper_id` is configured.
5. Send `X-App-Key` only for Subscriptions API calls and only if `app_key` is configured.
6. Send `X-Market-Id` for agreements/subscriptions calls when available.
7. Generate `X-Request-Id` (UUIDv4) for all **v2 mutating operations**.
8. Build a user agent string like:
   - `terraform-provider-godaddy/<version>`
   - append ` (<user_agent_suffix>)` if configured

### 5.4 Customer ID resolution rules

Some v2 endpoints require `customer_id`. The provider MUST resolve it as follows:

1. If `customer_id` is configured directly, use it.
2. Else if `shopper_id` is configured, call:
   - `GET /v1/shoppers/{shopperId}?includes=customerId`
3. Cache the resolved `customer_id` in provider runtime memory.
4. If neither `customer_id` nor `shopper_id` is available and a v2-only resource/data source is used, return a clear diagnostic.

Pseudo-code:

```go
func ResolveCustomerID(ctx context.Context) (string, diag.Diagnostics) {
    if p.config.CustomerID != "" {
        return p.config.CustomerID, nil
    }

    if p.runtime.cachedCustomerID != "" {
        return p.runtime.cachedCustomerID, nil
    }

    if p.config.ShopperID == "" {
        return "", diagError("customer_id required", "This operation uses a GoDaddy v2 endpoint and needs either provider.customer_id or provider.shopper_id.")
    }

    shopper, err := client.GetShopper(ctx, p.config.ShopperID, includes=["customerId"])
    if err != nil {
        return "", convertError(err)
    }
    if shopper.CustomerID == "" {
        return "", diagError("Unable to resolve customer_id", "GoDaddy did not return customerId for the configured shopper_id.")
    }

    p.runtime.cachedCustomerID = shopper.CustomerID
    return shopper.CustomerID, nil
}
```

---

## 6. Runtime architecture and repository layout

Start from the HashiCorp scaffolding framework shape [HC-SCAFFOLD], but split provider code from raw HTTP client code.

### 6.1 Required repository layout

```text
.
├── main.go
├── go.mod
├── go.sum
├── GNUmakefile
├── README.md
├── CHANGELOG.md
├── .goreleaser.yml
├── examples/
│   ├── provider/
│   ├── resources/
│   └── data-sources/
├── docs/
├── internal/
│   ├── provider/
│   │   ├── provider.go
│   │   ├── provider_config.go
│   │   ├── provider_schema.go
│   │   ├── data_source_domain.go
│   │   ├── data_source_domains.go
│   │   ├── data_source_dns_record_set.go
│   │   ├── data_source_domain_agreements.go
│   │   ├── data_source_domain_actions.go
│   │   ├── data_source_domain_forwarding.go
│   │   ├── data_source_shopper.go
│   │   ├── data_source_subscriptions.go              # optional later
│   │   ├── resource_dns_record_set.go
│   │   ├── resource_domain_settings.go
│   │   ├── resource_domain_nameservers.go
│   │   ├── resource_domain_contacts.go
│   │   ├── resource_domain_forwarding.go
│   │   ├── resource_domain_dnssec_records.go
│   │   ├── validators.go
│   │   ├── plan_modifiers.go
│   │   └── models_tf.go
│   ├── client/
│   │   ├── client.go
│   │   ├── auth.go
│   │   ├── transport.go
│   │   ├── rate_limit.go
│   │   ├── retry.go
│   │   ├── errors.go
│   │   ├── actions.go
│   │   ├── domains_v1.go
│   │   ├── domains_v2.go
│   │   ├── dns_v1.go
│   │   ├── shoppers_v1.go
│   │   ├── subscriptions_v1.go
│   │   └── models_api.go
│   ├── normalize/
│   │   ├── domain.go
│   │   ├── dns.go
│   │   ├── contacts.go
│   │   └── dnssec.go
│   └── acctest/
│       ├── provider_test.go
│       ├── testdata.go
│       ├── testdomain.go
│       └── random.go
└── tools/
```

### 6.2 Package responsibilities

- `internal/provider`
  - all Terraform-facing schema, models, resource/data source implementations, import parsing, diagnostics
- `internal/client`
  - GoDaddy HTTP client, request/response mapping, retries, pollers, typed errors
- `internal/normalize`
  - canonicalization and comparison helpers
- `internal/acctest`
  - test helpers and acceptance fixtures

### 6.3 Provider server entrypoint

Use the framework provider server in `main.go`. Keep version injection from build metadata.

---

## 7. HTTP client contract

### 7.1 Required client capabilities

The GoDaddy client layer MUST provide:

- endpoint resolution
- auth header injection
- optional `X-Shopper-Id`
- optional `X-App-Key`
- optional `X-Market-Id`
- optional `X-Request-Id`
- JSON encode/decode
- retry policy
- endpoint-aware rate limiting
- error parsing into typed Go errors
- async action polling
- request/response logging when `debug_http=true` without leaking secrets

### 7.2 Rate limiting

Because GoDaddy documents 60 requests per minute **per endpoint**, the provider MUST implement a conservative limiter.

Implementation rule:

- default effective rate: **50 rpm** per path template
- clamp user-configured `rate_limit_rpm` to `1..60`
- maintain a limiter keyed by normalized path template, e.g.:
  - `/v1/domains/{domain}`
  - `/v1/domains/{domain}/records/{type}/{name}`
  - `/v2/customers/{customerId}/domains/{domain}/actions/{type}`

A simple `x/time/rate` limiter per template is sufficient.

### 7.3 Retry policy

The client MUST retry:

- `429` using `retryAfterSec` from `ErrorLimit`
- transient `5xx` errors with exponential backoff and jitter
- transport-level connection resets / temporary network errors

Default retry policy:

- max retries: 5
- initial backoff: 1s
- multiplier: 2x
- max backoff: 16s
- jitter: +/- 20%

Do **not** retry:

- `400`, `401`, `403`, `404`, `409`, `422` unless a very specific endpoint says eventual consistency may be involved
- request bodies that are non-idempotent if the retry could duplicate side effects and there is no request ID guard

For v2 mutating calls, because we send `X-Request-Id`, retries are safer, but still do not blindly retry `409`.

### 7.4 Error parsing

Implement typed client errors.

Recommended internal types:

```go
type APIError struct {
    StatusCode int
    Code       string
    Message    string
    Fields     []APIErrorField
    RawBody    []byte
}

type APIErrorField struct {
    Path        string
    PathRelated string
    Code        string
    Message     string
}

type RateLimitError struct {
    APIError
    RetryAfterSec int
}
```

Terraform diagnostic mapping:

- `400` => configuration/request malformed; show `message` and field paths
- `401` => authentication error
- `403` => authorization / account not allowed / account not eligible
- `404` during `Read` => remove from state for real CRUD resources; for adopt-style resources remove from state if domain no longer exists
- `409` => conflict:
  - often import-required for create of already-existing remote object
  - sometimes domain not eligible or similar action in progress
- `422` => validation error; surface field-level details
- `429` => handled in retry loop; if exhausted, report rate limit error
- `500+` => retry first, then report server error

### 7.5 Partial success handling (HTTP 203 on v2 domain detail)

GoDaddy v2 domain detail may return `203 Partial Success` if some optional include sections could not be fully provided. The provider MUST treat this carefully:

- For **data sources**, return successfully with:
  - `partial = true`
  - warnings listing which optional include(s) were unavailable if determinable
  - missing sections set to `null`/empty
- For **managed resources**, do not rely on 203-prone include sections for critical refresh unless unavoidable.
- For `godaddy_domain_dnssec_records`, if the read path requires `dnssecRecords` and that section is unavailable, return an error rather than silently drifting.

---

## 8. Normalization and state canonicalization

Canonicalization is essential to avoid perpetual diffs.

### 8.1 Domain/FQDN normalization

Implement a helper that:

- trims whitespace
- lowercases
- trims a trailing dot
- converts Unicode domains to ASCII/Punycode with `idna.Lookup.ToASCII` (recommended)
- rejects empty strings

Use canonical lower-case strings in state.

### 8.2 DNS record-set normalization

For `godaddy_dns_record_set`:

- canonical `type` => uppercase
- canonical `name`:
  - accept `@` or empty from user/import
  - store `@` in Terraform state as the apex placeholder
- canonical record ordering:
  - sort records by tuple:
    - `data`
    - `priority`
    - `weight`
    - `port`
    - `protocol`
    - `service`
    - `ttl`
- trim trailing dot from CNAME/MX/NS target data only if GoDaddy returns/accepts it inconsistently
- preserve TXT string contents exactly; do not auto-quote or unquote

### 8.3 Nameserver normalization

- lowercase
- trim trailing dots
- deduplicate
- sort lexicographically in state

### 8.4 Contact normalization

- trim strings
- preserve user case for names/organization except where GoDaddy normalizes on read
- phone numbers: pass through as strings, do not aggressively reformat
- country: uppercase 2-letter ISO code
- optional fields absent => Terraform `null`

### 8.5 DNSSEC normalization

Canonicalize each record by:

- uppercase `algorithm`
- uppercase `digest_type`
- uppercase `flags`
- trim `digest`
- trim `public_key`
- represent absent optional numeric fields as `null`, not zero
- sort by tuple:
  - `key_tag`
  - `algorithm`
  - `digest_type`
  - `digest`
  - `flags`
  - `public_key`
  - `max_signature_life`

### 8.6 Import ID normalization

Import IDs MUST be canonicalized before writing state.

---

## 9. Resource and data source catalog

### 9.1 Managed resources in v1

Implement these resources:

1. `godaddy_dns_record_set`
2. `godaddy_domain_settings`
3. `godaddy_domain_nameservers`
4. `godaddy_domain_contacts`
5. `godaddy_domain_forwarding`
6. `godaddy_domain_dnssec_records`

### 9.2 Data sources in v1

Implement these data sources:

1. `godaddy_domain`
2. `godaddy_domains`
3. `godaddy_dns_record_set`
4. `godaddy_domain_agreements`
5. `godaddy_domain_actions`
6. `godaddy_domain_forwarding`
7. `godaddy_shopper`

### 9.3 Optional later surfaces

Later, but not required for first GA:

- `godaddy_subscriptions`
- `godaddy_subscription_product_groups`
- `godaddy_domain_notifications`
- `godaddy_domain_notification_schema`
- `godaddy_shopper_subaccount`
- Terraform actions for renew/register/transfer/privacy

---

## 10. Shared Terraform implementation conventions

Apply these conventions across all resources.

### 10.1 Schema conventions

- use nested attributes, not legacy block types, for new schemas
- explicit computed `id` on every resource
- use `UseStateForUnknown()` for `id` and other read-only computed attributes
- mark secrets/sensitive outputs as `Sensitive: true`
- use validators for enums, min/max items, and obvious shape checks
- prefer optional+computed only when read-after-write behavior requires it; otherwise keep schema strict

### 10.2 Import conventions

Support import for every resource.

Use these import ID formats:

- `godaddy_domain_settings` => `example.com`
- `godaddy_domain_nameservers` => `example.com`
- `godaddy_domain_contacts` => `example.com`
- `godaddy_domain_dnssec_records` => `example.com`
- `godaddy_dns_record_set` => `example.com,A,www`
- `godaddy_domain_forwarding` => `blog.example.com`

Parse composite IDs using comma separators, per [HC-IMPORT].

### 10.3 Read-after-write conventions

After every successful create/update/delete that changes remote state, do a fresh `Read` or equivalent GET and write canonical state.

### 10.4 Destroy semantics conventions

Not every GoDaddy-managed thing has a meaningful "delete" inverse.

Use these destroy rules:

- **True remote delete**
  - `godaddy_dns_record_set`
  - `godaddy_domain_forwarding`
  - `godaddy_domain_dnssec_records`
- **State-only unmanage**
  - `godaddy_domain_settings`
  - `godaddy_domain_nameservers`
  - `godaddy_domain_contacts`

For the state-only-unmanage resources, `Delete` MUST:

1. add a warning diagnostic explaining that Terraform management is being removed but the domain remains as-is remotely
2. remove the resource from state
3. not attempt to revert to unknown defaults

Provider docs SHOULD recommend:

```hcl
lifecycle {
  prevent_destroy = true
}
```

for these resources in production.

---

## 11. Detailed resource specification

---

### 11.1 `godaddy_dns_record_set`

#### 11.1.1 Purpose

Manage one DNS RRset for one domain / record type / name combination.

#### 11.1.2 Scope and restrictions

This resource MUST support only these types in v1:

- `A`
- `AAAA`
- `CNAME`
- `MX`
- `SRV`
- `TXT`

Do **not** support `NS` or `SOA` in this resource because:

- `NS` is separately managed by `godaddy_domain_nameservers`
- `SOA` delete is unsupported and should not be managed declaratively here

#### 11.1.3 Terraform schema

```hcl
resource "godaddy_dns_record_set" "example" {
  domain = "example.com"
  type   = "A"
  name   = "www" # default "@"

  records = [
    {
      data = "203.0.113.10"
      ttl  = 600
    }
  ]
}
```

##### Attributes

- `id` — computed string, format `domain,type,name`
- `domain` — required string, replace on change
- `type` — required string enum, replace on change
- `name` — optional string, default `@`, replace on change
- `records` — required set/list of nested objects
- `fqdn` — computed string
- `timeouts` — optional

##### `records` nested object fields

- `data` — required string for all supported types
- `ttl` — optional integer
- `priority` — required for `MX` and `SRV`
- `port` — required for `SRV`
- `protocol` — required for `SRV`
- `service` — required for `SRV`
- `weight` — required for `SRV`

Validation rules:

- `CNAME` MUST contain exactly one record
- `MX` records MUST all have `priority`
- `SRV` records MUST all have `priority`, `weight`, `port`, `protocol`, `service`, and `data`
- `A`, `AAAA`, `TXT`, `CNAME` MUST NOT set `priority`, `weight`, `port`, `protocol`, or `service`

#### 11.1.4 API mapping

- Read:
  - `GET /v1/domains/{domain}/records/{type}/{name}`
- Create / Update:
  - `PUT /v1/domains/{domain}/records/{type}/{name}`
- Delete:
  - `DELETE /v1/domains/{domain}/records/{type}/{name}`

#### 11.1.5 Create behavior

Create MUST be safe and must not silently take over an existing RRset.

Algorithm:

1. Normalize `domain`, `type`, `name`.
2. GET the RRset.
3. If the RRset exists and contains one or more records:
   - return an error instructing the user to import the resource
4. Else PUT the desired RRset.
5. Read back and set state.

Import-required create behavior is intentional and aligns with Terraform expectations for pre-existing remote objects.

#### 11.1.6 Read behavior

- If GoDaddy returns `404` or an empty array, remove the resource from state.
- Else canonicalize the records and store them.

#### 11.1.7 Update behavior

- PUT the full desired RRset.
- Read back and canonicalize.

#### 11.1.8 Delete behavior

- Call DELETE.
- Treat `404` as success.
- Remove from state.

#### 11.1.9 Import format

```bash
terraform import godaddy_dns_record_set.www example.com,A,www
terraform import godaddy_dns_record_set.apex_txt example.com,TXT,@
```

#### 11.1.10 State model

The resource state MUST represent the full RRset as returned by GoDaddy, normalized and sorted.

#### 11.1.11 Acceptance tests

Required tests:

- create/update/delete `A`
- create/import/update/delete `TXT` with multiple records
- create/update/delete `MX`
- create/update/delete `SRV`
- conflict on create when RRset already exists
- replace behavior when `domain`, `type`, or `name` changes

---

### 11.2 `godaddy_domain_settings`

#### 11.2.1 Purpose

Manage selected mutable settings on an existing domain without overlapping with nameservers or contacts.

#### 11.2.2 Ownership

This resource owns only:

- `locked`
- `renew_auto`
- `expose_registrant_organization`
- `expose_whois`

It MUST NOT own:

- `name_servers`
- contacts
- DNS records
- DNSSEC
- forwarding
- auth code
- transfer lifecycle

#### 11.2.3 Terraform schema

```hcl
resource "godaddy_domain_settings" "example" {
  domain = "example.com"

  locked                        = true
  renew_auto                    = true
  expose_registrant_organization = false
  expose_whois                  = false

  # required only when turning exposure flags on
  consent = {
    agreed_by      = "203.0.113.50"
    agreed_at      = "2026-04-08T10:30:00Z"
    agreement_keys = ["EXPOSE_WHOIS"]
  }
}
```

##### Attributes

- `id` — computed string, format `example.com`
- `domain` — required string, replace on change
- `locked` — optional bool
- `renew_auto` — optional bool
- `expose_registrant_organization` — optional bool
- `expose_whois` — optional bool
- `consent` — optional nested object
- computed read-only:
  - `status`
  - `created_at`
  - `expires_at`
  - `renew_deadline`
  - `privacy`
  - `transfer_protected`
  - `expiration_protected`
  - `hold_registrar`
  - `name_servers`

##### `consent` nested object

- `agreed_by` — required string when block present
- `agreed_at` — required RFC3339 timestamp string when block present
- `agreement_keys` — required set(string) when block present

Valid values for `agreement_keys`:

- `EXPOSE_REGISTRANT_ORGANIZATION`
- `EXPOSE_WHOIS`

#### 11.2.4 API mapping

- Read:
  - `GET /v1/domains/{domain}`
- Update:
  - `PATCH /v1/domains/{domain}` with `DomainUpdate`

#### 11.2.5 Create behavior

This is an **adopt-and-reconcile** resource over an existing domain.

Algorithm:

1. GET domain via v1.
2. If not found, return error: domain must already exist.
3. Compute desired patch from configured fields only.
4. If no changes needed, write state from current read.
5. Else PATCH.
6. Read back and write state.

#### 11.2.6 Consent validation behavior

Implement plan-time validation:

- If `expose_registrant_organization` transitions from `false`/unknown remote value to `true`, require consent with `EXPOSE_REGISTRANT_ORGANIZATION`.
- If `expose_whois` transitions from `false`/unknown remote value to `true`, require consent with `EXPOSE_WHOIS`.

Use prior state + planned values in `ModifyPlan` to determine whether consent is required.

#### 11.2.7 Update behavior

PATCH only the owned fields that are set in plan.

#### 11.2.8 Delete behavior

State-only unmanage. Do not attempt remote deletion or reversion.

#### 11.2.9 Import format

```bash
terraform import godaddy_domain_settings.example example.com
```

#### 11.2.10 Acceptance tests

Required tests:

- import existing domain
- toggle `locked`
- toggle `renew_auto`
- enable `expose_whois` with consent
- validation error when exposure flags are enabled without required consent
- state-only destroy behavior

---

### 11.3 `godaddy_domain_nameservers`

#### 11.3.1 Purpose

Manage the authoritative nameserver set for an existing domain.

#### 11.3.2 Why this is separate

Nameservers are a full-set replacement concern and operationally sensitive. They must not overlap with `godaddy_domain_settings` or DNS RRset resources.

#### 11.3.3 Terraform schema

```hcl
resource "godaddy_domain_nameservers" "example" {
  domain = "example.com"

  name_servers = [
    "ns1.example.net",
    "ns2.example.net",
  ]
}
```

##### Attributes

- `id` — computed string, format `example.com`
- `domain` — required string, replace on change
- `name_servers` — required set/list(string), min 2
- computed:
  - `status`
  - `updated_via_v2` — computed bool for diagnostics/debugging only (optional)
- `timeouts` — optional

#### 11.3.4 API mapping

Preferred path:

- `PUT /v2/customers/{customerId}/domains/{domain}/nameServers`
- poll `GET /v2/customers/{customerId}/domains/{domain}/actions/DOMAIN_UPDATE_NAME_SERVERS`

Fallback path when `customer_id` unavailable:

- `PATCH /v1/domains/{domain}` with `nameServers`

Read path:

- `GET /v1/domains/{domain}`

#### 11.3.5 Create behavior

Adopt-and-reconcile existing domain.

Algorithm:

1. GET domain.
2. If absent, error.
3. Compare current normalized nameservers to desired.
4. If already equal, write state.
5. Else:
   - if `customer_id` available, use v2 PUT + action polling
   - else use v1 PATCH
6. Read back and write state.

#### 11.3.6 Update behavior

Same as create, but in-place.

#### 11.3.7 Delete behavior

State-only unmanage.

#### 11.3.8 Special error behavior

Surface clear errors for:

- fewer than two nameservers
- domain not eligible to change nameservers
- nameserver update requiring 2FA not supported by the API
- conflicting action already in progress

#### 11.3.9 Import format

```bash
terraform import godaddy_domain_nameservers.example example.com
```

#### 11.3.10 Acceptance tests

Required tests:

- import + no-op
- update nameservers through v2 path when customer_id available
- fallback v1 path if no customer_id (testable in unit tests if not practical in acceptance)
- validation error for fewer than two nameservers
- state-only destroy behavior

---

### 11.4 `godaddy_domain_contacts`

#### 11.4.1 Purpose

Manage the domain's registrant/admin/tech/billing contacts as a single owned set.

#### 11.4.2 Design choice

This resource MUST manage the **full contact set**. Partial-contact ownership is forbidden in v1 because it would introduce ambiguity and overlap.

#### 11.4.3 Terraform schema

```hcl
resource "godaddy_domain_contacts" "example" {
  domain = "example.com"

  registrant = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  admin = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  # tech and billing use the same shape as registrant/admin

  # optional v2-only enhancement
  identity_document_id = "abc123"
}
```

##### Attributes

- `id` — computed string, format `example.com`
- `domain` — required string, replace on change
- `registrant` — required nested object
- `admin` — required nested object
- `tech` — required nested object
- `billing` — required nested object
- `identity_document_id` — optional string
- `timeouts` — optional

##### Contact nested object fields

- `name_first` — required
- `name_middle` — optional
- `name_last` — required
- `organization` — optional
- `job_title` — optional
- `email` — required
- `phone` — required
- `fax` — optional
- `address_mailing` — required nested object

##### `address_mailing` fields

- `address1` — required
- `address2` — optional
- `city` — required
- `state` — required
- `postal_code` — required
- `country` — required 2-letter ISO code

#### 11.4.4 API mapping

Default path:

- `PATCH /v1/domains/{domain}/contacts` with `DomainContacts`

v2 path when `identity_document_id` is set:

- `PATCH /v2/customers/{customerId}/domains/{domain}/contacts`
- poll `GET /v2/customers/{customerId}/domains/{domain}/actions/DOMAIN_UPDATE_CONTACTS`

Read path:

- `GET /v1/domains/{domain}` for default readback
- if v2-specific data is needed later, v2 can be used internally, but state shape MUST remain the same

#### 11.4.5 Create behavior

Adopt-and-reconcile existing domain contacts.

Algorithm:

1. GET domain detail.
2. If domain absent, error.
3. Compare remote contacts to desired normalized contacts.
4. If equal, write state.
5. Else:
   - if `identity_document_id` is set, resolve `customer_id`, use v2 PATCH + poll
   - else use v1 PATCH
6. Read back and write state.

#### 11.4.6 Update behavior

Same as create, in-place.

#### 11.4.7 Delete behavior

State-only unmanage.

#### 11.4.8 Import format

```bash
terraform import godaddy_domain_contacts.example example.com
```

#### 11.4.9 Acceptance tests

Required tests:

- import + no-op
- update all four contacts via v1 path
- update registrant with `identity_document_id` via v2 path (only when test account/domain supports it)
- state-only destroy behavior

---

### 11.5 `godaddy_domain_forwarding`

#### 11.5.1 Purpose

Manage GoDaddy URL forwarding for one FQDN.

This is **web URL forwarding**, not privacy-email forwarding.

#### 11.5.2 Prerequisite

This resource requires `customer_id` (direct or resolved).

#### 11.5.3 Terraform schema

```hcl
resource "godaddy_domain_forwarding" "blog" {
  fqdn = "blog.example.com"
  type = "REDIRECT_PERMANENT"
  url  = "https://www.example.com/blog"

  mask = {
    title       = "Example Blog"
    description = "Blog redirect"
    keywords    = "blog,example"
  }
}
```

##### Attributes

- `id` — computed string, format `blog.example.com`
- `fqdn` — required string, replace on change
- `type` — required enum:
  - `MASKED`
  - `REDIRECT_PERMANENT`
  - `REDIRECT_TEMPORARY`
- `url` — required string
- `mask` — optional nested object:
  - `title`
  - `description`
  - `keywords`

#### 11.5.4 API mapping

- Create:
  - `POST /v2/customers/{customerId}/domains/forwards/{fqdn}`
- Read:
  - `GET /v2/customers/{customerId}/domains/forwards/{fqdn}`
- Update:
  - `PUT /v2/customers/{customerId}/domains/forwards/{fqdn}`
- Delete:
  - `DELETE /v2/customers/{customerId}/domains/forwards/{fqdn}`

#### 11.5.5 Create behavior

Algorithm:

1. Resolve `customer_id`.
2. GET forwarding for `fqdn`.
3. If forwarding already exists:
   - return import-required error
4. Else POST create body.
5. GET and write state.

Do not silently adopt existing forwarding on create.

#### 11.5.6 Read behavior

- `404` => remove from state
- else map returned `DomainForwarding` to state

#### 11.5.7 Update behavior

- PUT full desired object
- GET and write state

#### 11.5.8 Delete behavior

- DELETE
- `404` => success
- remove from state

#### 11.5.9 Import format

```bash
terraform import godaddy_domain_forwarding.blog blog.example.com
```

#### 11.5.10 Acceptance tests

Required tests:

- create/import/update/delete
- create conflict when forwarding already exists
- masked forwarding
- temporary vs permanent redirect variants

---

### 11.6 `godaddy_domain_dnssec_records`

#### 11.6.1 Purpose

Manage the complete DNSSEC record set for a domain.

#### 11.6.2 Design choice

This resource manages the **entire set**, not a single record. That matches Terraform ownership better and avoids partial-set drift.

#### 11.6.3 Terraform schema

```hcl
resource "godaddy_domain_dnssec_records" "example" {
  domain = "example.com"

  records = [
    {
      key_tag           = 12345
      algorithm         = "RSASHA256"
      digest_type       = "SHA256"
      digest            = "ABCDEF1234567890"
      flags             = "KSK"
      public_key        = "BASE64VALUE"
      max_signature_life = 86400
    }
  ]
}
```

##### Attributes

- `id` — computed string, format `example.com`
- `domain` — required string, replace on change
- `records` — required set/list of nested objects
- `timeouts` — optional

##### DNSSEC record fields

The provider MUST require these fields even though the raw GoDaddy schema only marks `algorithm` mandatory:

- `key_tag` — required integer
- `algorithm` — required enum
- `digest_type` — required enum
- `digest` — required string

Optional:

- `flags` — optional enum `ZSK` / `KSK`
- `public_key` — optional string
- `max_signature_life` — optional positive integer

This stronger schema is intentional so that Terraform has enough information to identify records deterministically.

#### 11.6.4 API mapping

- add records:
  - `PATCH /v2/customers/{customerId}/domains/{domain}/dnssecRecords`
- remove records:
  - `DELETE /v2/customers/{customerId}/domains/{domain}/dnssecRecords`
- read:
  - `GET /v2/customers/{customerId}/domains/{domain}?includes=dnssecRecords`
- poll actions:
  - `GET /v2/customers/{customerId}/domains/{domain}/actions/DNSSEC_CREATE`
  - `GET /v2/customers/{customerId}/domains/{domain}/actions/DNSSEC_DELETE`

#### 11.6.5 Create behavior

Algorithm:

1. Resolve `customer_id`.
2. Read current DNSSEC set via v2 domain detail include.
3. If current set is non-empty:
   - return import-required error
4. Else PATCH add all desired records.
5. Poll `DNSSEC_CREATE`.
6. Read back and write state.

#### 11.6.6 Update behavior

Algorithm:

1. Read current set.
2. Compute:
   - `to_add = desired - current`
   - `to_remove = current - desired`
3. If both empty, no-op.
4. If `to_add` non-empty:
   - PATCH add
   - poll `DNSSEC_CREATE`
5. If `to_remove` non-empty:
   - DELETE remove
   - poll `DNSSEC_DELETE`
6. Read back and write state.

**Important:** Add before remove. This is safer for DNSSEC rollover scenarios.

#### 11.6.7 Read behavior

- If domain missing => remove from state
- If domain exists but `dnssecRecords` is empty => remove resource from state
- If v2 returns partial success without `dnssecRecords`, return error

#### 11.6.8 Delete behavior

- Read current DNSSEC set
- If empty => success
- DELETE all current records
- poll `DNSSEC_DELETE`
- remove from state

#### 11.6.9 Import format

```bash
terraform import godaddy_domain_dnssec_records.example example.com
```

#### 11.6.10 Acceptance tests

Required tests:

- import/create/update/delete
- add-then-remove rollover sequence
- empty remote set removes state
- action polling and timeout handling

---

## 12. Detailed data source specification

---

### 12.1 `godaddy_domain`

#### Purpose

Return domain details for one domain. Use v1 for basic fields and v2 only when advanced includes are requested.

#### Schema

Inputs:

- `domain` — required
- `include_auth_code` — optional bool, default false
- `include_actions` — optional bool, default false
- `include_dnssec_records` — optional bool, default false
- `include_registry_status_codes` — optional bool, default false

Outputs (basic):

- `domain`
- `domain_id`
- `status`
- `created_at`
- `expires_at`
- `renew_auto`
- `renew_deadline`
- `locked`
- `privacy`
- `transfer_protected`
- `expiration_protected`
- `hold_registrar`
- `name_servers`
- `contacts`
- `expose_registrant_organization`
- `expose_whois`
- `auth_code` — sensitive, only when requested
- advanced optional outputs:
  - `actions`
  - `dnssec_records`
  - `registry_status_codes`
- `partial` — bool

#### Read rules

- if no advanced include requested, use `GET /v1/domains/{domain}`
- if advanced include requested:
  - resolve `customer_id`
  - call `GET /v2/customers/{customerId}/domains/{domain}?includes=...`
- if v2 returns 203:
  - set `partial = true`
  - return what is available
  - add warning diagnostic
- `auth_code` MUST be sensitive and omitted unless requested

---

### 12.2 `godaddy_domains`

#### Purpose

List domains for the current shopper.

#### Schema

Inputs:

- `statuses` — optional list(string)
- `status_groups` — optional list(string)
- `limit` — optional int, default 100, max 1000
- `marker` — optional string
- `includes` — optional list(enum: `authCode`, `contacts`, `nameServers`)
- `modified_date` — optional RFC3339 string

Outputs:

- `domains` — list of domain summary objects

#### API mapping

- `GET /v1/domains`

#### Notes

This data source MAY fetch one page only. Keep behavior faithful to GoDaddy's `limit`/`marker` contract rather than inventing undocumented pagination semantics.

---

### 12.3 `godaddy_dns_record_set`

#### Purpose

Read a DNS RRset without managing it.

#### Schema

Inputs:

- `domain`
- `type`
- `name` (default `@`)

Outputs:

- `records`
- `fqdn`

#### API mapping

- `GET /v1/domains/{domain}/records/{type}/{name}`

This data source MAY support all readable GoDaddy record types, including `NS` and `SOA`, because it is read-only.

---

### 12.4 `godaddy_domain_agreements`

#### Purpose

Retrieve legal agreements required for domain operations.

#### Schema

Inputs:

- `tlds` — required list(string)
- `privacy` — required bool
- `for_transfer` — optional bool

Outputs:

- `agreements` — list of:
  - `agreement_key`
  - `title`
  - `content`
  - `url`

#### API mapping

- `GET /v1/domains/agreements`

Send `X-Market-Id` when configured.

---

### 12.5 `godaddy_domain_actions`

#### Purpose

Inspect recent v2 actions for a domain.

#### Prerequisite

Requires `customer_id`.

#### Schema

Inputs:

- `domain` — required

Outputs:

- `actions` — list of action objects:
  - `type`
  - `origination`
  - `created_at`
  - `started_at`
  - `completed_at`
  - `modified_at`
  - `status`
  - `request_id`
  - `reason`:
    - `code`
    - `message`
    - `fields`

#### API mapping

- `GET /v2/customers/{customerId}/domains/{domain}/actions`

Optional enhancement: add an input `type` to additionally fetch `GET /actions/{type}` when requested.

---

### 12.6 `godaddy_domain_forwarding`

#### Purpose

Read forwarding config for one FQDN.

#### Prerequisite

Requires `customer_id`.

#### Schema

Inputs:

- `fqdn` — required
- `include_subs` — optional bool, default false

Outputs:

- `fqdn`
- `type`
- `url`
- `mask`
- `subs` — optional if GoDaddy returns it with `includeSubs=true`

#### API mapping

- `GET /v2/customers/{customerId}/domains/forwards/{fqdn}`

---

### 12.7 `godaddy_shopper`

#### Purpose

Read shopper details and optionally resolve `customer_id`.

#### Schema

Inputs:

- `shopper_id` — required
- `include_customer_id` — optional bool, default true

Outputs:

- shopper details from GoDaddy
- `customer_id` when available

#### API mapping

- `GET /v1/shoppers/{shopperId}?includes=customerId`

---

## 13. Async action polling specification

Some v2 operations are asynchronous. The provider MUST implement a reusable poller.

### 13.1 Action status model

The GoDaddy action object includes:

- `type`
- `origination`
- `createdAt`
- `startedAt`
- `completedAt`
- `modifiedAt`
- `status`
- `reason`
- `requestId`

Recognized statuses:

- `ACCEPTED`
- `AWAITING`
- `CANCELLED`
- `FAILED`
- `PENDING`
- `SUCCESS`

### 13.2 Poller API

Implement an internal helper:

```go
func PollDomainAction(
    ctx context.Context,
    customerID string,
    domain string,
    actionType string,
    requestID string,
    timeout time.Duration,
) (*Action, error)
```

### 13.3 Poll algorithm

```go
deadline := time.Now().Add(timeout)

for {
    if time.Now().After(deadline) {
        return nil, ErrTimeout
    }

    action, err := client.GetDomainAction(ctx, customerID, domain, actionType)
    if err != nil {
        if is404EventuallyConsistent(err) {
            sleep(pollInterval)
            continue
        }
        return nil, err
    }

    // If requestID is set and action request IDs do not match, try list lookup.
    if requestID != "" && action.RequestID != "" && action.RequestID != requestID {
        actions, listErr := client.ListDomainActions(ctx, customerID, domain)
        if listErr == nil {
            action = findMatchingAction(actions, actionType, requestID)
        }
        if action == nil {
            sleep(pollInterval)
            continue
        }
    }

    switch action.Status {
    case "SUCCESS":
        return action, nil
    case "FAILED", "CANCELLED":
        return action, ErrActionFailed{Action: action}
    case "AWAITING":
        return action, ErrActionAwaitingInput{Action: action}
    case "ACCEPTED", "PENDING":
        sleep(pollInterval)
        continue
    default:
        sleep(pollInterval)
        continue
    }
}
```

### 13.4 Resources that must use the poller

- `godaddy_domain_nameservers` (v2 path)
- `godaddy_domain_contacts` when using v2
- `godaddy_domain_dnssec_records`

### 13.5 Poll timeout defaults

Use resource timeouts if set; otherwise:

- create: 10 minutes
- update: 10 minutes
- delete: 10 minutes

---

## 14. Endpoint mapping appendix

This section is intentionally implementation-oriented.

### 14.1 Core v1 endpoints

- `GET /v1/domains`
- `GET /v1/domains/agreements`
- `GET /v1/domains/{domain}`
- `PATCH /v1/domains/{domain}`
- `PATCH /v1/domains/{domain}/contacts`
- `GET /v1/domains/{domain}/records/{type}/{name}`
- `PUT /v1/domains/{domain}/records/{type}/{name}`
- `DELETE /v1/domains/{domain}/records/{type}/{name}`

### 14.2 Core v2 endpoints

- `GET /v2/customers/{customerId}/domains/{domain}`
- `PUT /v2/customers/{customerId}/domains/{domain}/nameServers`
- `PATCH /v2/customers/{customerId}/domains/{domain}/contacts`
- `PATCH /v2/customers/{customerId}/domains/{domain}/dnssecRecords`
- `DELETE /v2/customers/{customerId}/domains/{domain}/dnssecRecords`
- `GET /v2/customers/{customerId}/domains/{domain}/actions`
- `GET /v2/customers/{customerId}/domains/{domain}/actions/{type}`
- `DELETE /v2/customers/{customerId}/domains/{domain}/actions/{type}`
- `GET /v2/customers/{customerId}/domains/forwards/{fqdn}`
- `POST /v2/customers/{customerId}/domains/forwards/{fqdn}`
- `PUT /v2/customers/{customerId}/domains/forwards/{fqdn}`
- `DELETE /v2/customers/{customerId}/domains/forwards/{fqdn}`

### 14.3 Shopper endpoints

- `GET /v1/shoppers/{shopperId}?includes=customerId`
- optional later:
  - `POST /v1/shoppers/subaccount`

### 14.4 Subscription endpoints (optional later)

- `GET /v1/subscriptions`
- `GET /v1/subscriptions/productGroups`
- `GET /v1/subscriptions/{subscriptionId}`
- `DELETE /v1/subscriptions/{subscriptionId}`

---

## 15. API model mapping appendix

### 15.1 `DomainUpdate` mapping

Terraform `godaddy_domain_settings` fields map to `DomainUpdate` JSON:

- `locked` -> `locked`
- `renew_auto` -> `renewAuto`
- `expose_registrant_organization` -> `exposeRegistrantOrganization`
- `expose_whois` -> `exposeWhois`
- `consent.agreed_by` -> `consent.agreedBy`
- `consent.agreed_at` -> `consent.agreedAt`
- `consent.agreement_keys` -> `consent.agreementKeys`

### 15.2 `DomainContacts` (v1) mapping

Terraform role block -> `DomainContacts` JSON:

- `registrant` -> `contactRegistrant`
- `admin` -> `contactAdmin`
- `tech` -> `contactTech`
- `billing` -> `contactBilling`

Each contact field maps:

- `name_first` -> `nameFirst`
- `name_middle` -> `nameMiddle`
- `name_last` -> `nameLast`
- `organization` -> `organization`
- `job_title` -> `jobTitle`
- `email` -> `email`
- `phone` -> `phone`
- `fax` -> `fax`
- `address_mailing.address1` -> `addressMailing.address1`
- `address_mailing.address2` -> `addressMailing.address2`
- `address_mailing.city` -> `addressMailing.city`
- `address_mailing.state` -> `addressMailing.state`
- `address_mailing.postal_code` -> `addressMailing.postalCode`
- `address_mailing.country` -> `addressMailing.country`

### 15.3 `DNSRecord` mapping

Terraform nested record -> GoDaddy `DNSRecord`:

- `data` -> `data`
- `ttl` -> `ttl`
- `priority` -> `priority`
- `port` -> `port`
- `protocol` -> `protocol`
- `service` -> `service`
- `weight` -> `weight`

Do **not** persist `type` or `name` inside each nested record; those are owned by the parent resource identity.

### 15.4 `DomainForwardingCreate` mapping

Terraform:

- `type` -> `type`
- `url` -> `url`
- `mask.title` -> `mask.title`
- `mask.description` -> `mask.description`
- `mask.keywords` -> `mask.keywords`

### 15.5 `DomainDnssec` mapping

Terraform:

- `algorithm` -> `algorithm`
- `key_tag` -> `keyTag`
- `digest_type` -> `digestType`
- `digest` -> `digest`
- `flags` -> `flags`
- `public_key` -> `publicKey`
- `max_signature_life` -> `maxSignatureLife`

---

## 16. Terraform framework implementation details

### 16.1 Provider implementation checklist

The provider MUST implement:

- `provider.Provider`
- `Metadata`
- `Schema`
- `Configure`
- `Resources`
- `DataSources`

Later, if actions are added:

- `provider.ProviderWithActions`

### 16.2 Resource implementation checklist

Each resource MUST implement:

- `resource.Resource`
- `Metadata`
- `Schema`
- `Configure`
- `Create`
- `Read`
- `Update`
- `Delete`

And where needed:

- `resource.ResourceWithImportState`
- `ModifyPlan`
- `ConfigValidators`
- `ResourceWithConfigure`
- timeouts support

### 16.3 Data source implementation checklist

Each data source MUST implement:

- `datasource.DataSource`
- `Metadata`
- `Schema`
- `Configure`
- `Read`

### 16.4 Plan modifiers

Use plan modifiers for:

- `id` => `UseStateForUnknown`
- read-only computed fields => `UseStateForUnknown`
- identity fields (`domain`, `type`, `name`, `fqdn`) => `RequiresReplace`

### 16.5 Sensitive data rules

- provider `api_key`, `api_secret`, `app_key` => sensitive
- data source `auth_code` => sensitive
- do not expose `authCode` in managed resource state
- do not log auth headers or sensitive bodies

### 16.6 Write-only arguments

Write-only arguments are not necessary for the MVP resources in this provider. Avoid them unless a later action/resource must accept a secret that should never persist in state.

---

## 17. Testing specification

### 17.1 Unit tests required

Implement unit tests for:

- base URL resolution
- auth header injection
- `customer_id` resolution via shopper lookup
- rate limiter behavior
- retry policy
- typed error parsing
- action polling success / failure / timeout
- DNS normalization
- nameserver normalization
- contact mapping
- DNSSEC diff calculation
- import ID parsing

### 17.2 Acceptance test environment variables

Use these environment variables for acceptance tests:

- `GODADDY_API_KEY`
- `GODADDY_API_SECRET`
- `GODADDY_ENDPOINT`
- `GODADDY_SHOPPER_ID` (optional)
- `GODADDY_CUSTOMER_ID` (optional but recommended)
- `GODADDY_APP_KEY` (optional, subscriptions only)
- `GODADDY_TEST_DOMAIN`
- `GODADDY_TEST_FORWARD_FQDN` (optional)
- `GODADDY_TEST_FORWARD_URL` (optional)

### 17.3 Acceptance testing rules

- acceptance tests MUST run only when `TF_ACC=1`
- tests MUST use a dedicated low-value test domain
- DNS tests MUST use disposable hostnames (random prefixes) and not the apex unless specifically intended
- tests that mutate nameservers should be in a separate suite because of higher operational risk
- tests SHOULD be serialized per domain to avoid collisions and rate-limit issues

### 17.4 Acceptance test cases by resource

#### `godaddy_dns_record_set`

- create/read/update/delete A
- create/import/update/delete TXT
- MX with priorities
- SRV with full SRV shape
- already-exists conflict requires import

#### `godaddy_domain_settings`

- import existing domain
- toggle lock
- toggle auto-renew
- consent validation for WHOIS exposure
- state-only delete

#### `godaddy_domain_nameservers`

- import existing domain
- update nameservers
- state-only delete
- v2 async polling behavior

#### `godaddy_domain_contacts`

- import existing domain
- update contacts
- optional v2 identity document path
- state-only delete

#### `godaddy_domain_forwarding`

- create/import/update/delete
- masked forwarding
- conflict requires import

#### `godaddy_domain_dnssec_records`

- create/import/update/delete
- add-before-remove rollover
- remove state when remote set becomes empty

### 17.5 Manual test matrix

Before first release, manually verify on a real eligible production account:

- provider auth
- DNS RRset CRUD
- domain settings toggles
- nameserver update on a non-protected test domain
- forwarding CRUD
- DNSSEC CRUD if domain/TLD supports it

---

## 18. Documentation and release specification

### 18.1 Generated docs

Use `tfplugindocs` to generate Registry docs.

Requirements:

- every provider/resource/data source schema field must have `MarkdownDescription`
- examples must exist under `examples/`
- docs generation must be part of `make docs`

### 18.2 Required examples

Ship at least these example configurations:

- provider authentication
- DNS RRset
- domain settings
- nameservers
- contacts
- forwarding
- DNSSEC
- domain data source
- agreements data source

### 18.3 Build and release

Use `goreleaser` to publish provider binaries for:

- linux amd64/arm64
- darwin amd64/arm64
- windows amd64/arm64

Include:

- semantic version tags
- changelog entries
- registry manifest/package layout expected by Terraform Registry

### 18.4 Make targets

Recommended `GNUmakefile` targets:

- `make build`
- `make test`
- `make testacc`
- `make lint`
- `make fmt`
- `make docs`
- `make install`

---

## 19. Phased delivery plan

### Phase 0 — Project bootstrap

Deliver:

- scaffolded provider repo
- provider config schema
- client skeleton
- docs generation pipeline
- CI basics

Exit criteria:

- provider builds
- `terraform providers schema -json` works
- docs generation works

### Phase 1 — Core reads + DNS

Deliver:

- `godaddy_domain`
- `godaddy_domains`
- `godaddy_dns_record_set` data source
- `godaddy_dns_record_set` resource
- import support
- unit tests + acceptance for DNS

Exit criteria:

- DNS RRset lifecycle stable and idempotent
- import works
- no perpetual diffs

### Phase 2 — Existing-domain settings

Deliver:

- `godaddy_domain_settings`
- `godaddy_domain_nameservers`
- `godaddy_domain_contacts`
- `godaddy_domain_agreements`
- `godaddy_shopper`
- action poller
- acceptance coverage

Exit criteria:

- settings resources can import/adopt existing domains
- state-only delete semantics documented and tested

### Phase 3 — Forwarding + DNSSEC

Deliver:

- `godaddy_domain_forwarding`
- `godaddy_domain_actions`
- `godaddy_domain_dnssec_records`

Exit criteria:

- async v2 mutation/polling stable
- forwarding CRUD stable
- DNSSEC diff/reconcile stable

### Phase 4 — Optional reseller/subscription surfaces

Deliver only if needed:

- `godaddy_subscriptions`
- `godaddy_subscription_product_groups`
- `godaddy_shopper_subaccount`

### Phase 5 — Optional Terraform actions

Later actions MAY include:

- `godaddy_domain_renew`
- `godaddy_domain_register`
- `godaddy_domain_transfer`
- `godaddy_subscription_cancel`

These should be actions, not CRUD resources.

---

## 20. Definition of done

The provider is "finished" for its first release when all of the following are true:

1. It authenticates successfully against GoDaddy OTE and Production.
2. It supports all six core resources defined in this document.
3. It supports all seven v1 data sources defined in this document.
4. Import works for every resource.
5. DNS resources are RRset-based and do not produce perpetual diffs.
6. Domain settings ownership is split across non-overlapping resources.
7. Async action polling works for v2 mutations.
8. Rate limiting and retries are implemented and covered by tests.
9. Docs and examples are generated and accurate.
10. Acceptance tests pass in a real GoDaddy environment with `TF_ACC=1`.
11. The provider can be built and released for Terraform Registry distribution.

---

## 21. Example end-user configuration

### 21.1 Provider

```hcl
terraform {
  required_providers {
    godaddy = {
      source  = "acme/godaddy"
      version = "~> 0.1"
    }
  }
}

provider "godaddy" {
  api_key     = var.godaddy_api_key
  api_secret  = var.godaddy_api_secret
  endpoint    = "production"
  customer_id = var.godaddy_customer_id
}
```

### 21.2 DNS RRset

```hcl
resource "godaddy_dns_record_set" "www_a" {
  domain = "example.com"
  type   = "A"
  name   = "www"

  records = [
    {
      data = "203.0.113.10"
      ttl  = 600
    }
  ]
}
```

### 21.3 Domain settings

```hcl
resource "godaddy_domain_settings" "example" {
  domain     = "example.com"
  locked     = true
  renew_auto = true
}
```

### 21.4 Nameservers

```hcl
resource "godaddy_domain_nameservers" "example" {
  domain = "example.com"
  name_servers = [
    "ns1.example.net",
    "ns2.example.net",
  ]
}
```

### 21.5 Contacts

```hcl
resource "godaddy_domain_contacts" "example" {
  domain = "example.com"

  registrant = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  admin = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  tech = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  billing = {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"
    address_mailing = {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### 21.6 Forwarding

```hcl
resource "godaddy_domain_forwarding" "blog" {
  fqdn = "blog.example.com"
  type = "REDIRECT_PERMANENT"
  url  = "https://www.example.com/blog"
}
```

### 21.7 DNSSEC

```hcl
resource "godaddy_domain_dnssec_records" "example" {
  domain = "example.com"

  records = [
    {
      key_tag            = 12345
      algorithm          = "RSASHA256"
      digest_type        = "SHA256"
      digest             = "ABCDEF1234567890"
      flags              = "KSK"
      max_signature_life = 86400
    }
  ]
}
```

### 21.8 Data source

```hcl
data "godaddy_domain" "example" {
  domain                = "example.com"
  include_dnssec_records = true
}
```

---

## 22. Optional future action specification (not in MVP)

If actions are implemented later, use Terraform actions because actions are explicitly intended for day-two operations and do not affect resource state.

Candidate action types:

- `godaddy_domain_renew`
- `godaddy_domain_register`
- `godaddy_domain_transfer`
- `godaddy_subscription_cancel`

General rules:

- actions MUST be opt-in and documented as billing-affecting
- actions MUST send `X-Request-Id`
- actions MUST poll GoDaddy action status where applicable
- actions MUST not be disguised as normal CRUD resources

---

## 23. Final implementation notes for the coding agent

1. Build the provider exactly around **non-overlapping ownership**.
2. Do not introduce a `godaddy_dns_record` resource.
3. Do not store transfer auth codes in managed resource state.
4. Do not silently adopt pre-existing forwarding, DNS RRsets, or DNSSEC sets on create; require import.
5. For singleton existing-domain resources (`settings`, `nameservers`, `contacts`), create means **adopt and reconcile**; delete means **unmanage**.
6. Keep the client hand-written, small, and well-tested.
7. Read back after every mutation and canonicalize state aggressively.
8. Implement clear diagnostics for consent, rate limits, protected-domain restrictions, and import-required conflicts.
9. Ship examples and docs with the first usable commit, not as an afterthought.

---

## 24. Source notes used to derive this specification

This specification is derived from these current documented facts:

- GoDaddy documents the auth header format, separate OTE and Production environments, 60 rpm endpoint limits, account eligibility restrictions for some production Domains APIs, Good as Gold requirements for purchases, and self-serve vs reseller behavior with `X-Shopper-Id`. See [GD-GETSTARTED].
- GoDaddy documents the Domains, Shoppers, and Subscriptions API surfaces and publishes the Swagger documents for each. See [GD-DOMAINS-DOC], [GD-SHOPPERS-DOC], [GD-SUBSCRIPTIONS-DOC].
- GoDaddy's Domains Swagger documents:
  - v1 DNS RRset endpoints,
  - v1 domain detail/update,
  - v1 domain contacts,
  - v2 domain detail includes,
  - v2 nameserver update,
  - v2 contacts update,
  - v2 forwarding,
  - v2 DNSSEC,
  - v2 domain actions and action statuses,
  - consent requirements for `exposeRegistrantOrganization` and `exposeWhois`,
  - and the note that some protected/high-value nameserver changes may require 2FA not supported via API. See [GD-DOMAINS-SWAGGER].
- GoDaddy's Shoppers Swagger documents subaccount creation, shopper/customer ID distinction, shopper lookup with `customerId`, OTE limits on shopper deletion, and reseller password behavior. See [GD-SHOPPERS-SWAGGER].
- GoDaddy's Subscriptions Swagger documents `X-App-Key`, list/product-group/detail, and subscription cancel. See [GD-SUBSCRIPTIONS-SWAGGER].
- HashiCorp documents the Plugin Framework as the recommended way to build new providers, the resource CRUD assumptions, import conventions, actions, acceptance testing with `TF_ACC=1`, docs generation with `tfplugindocs`, and optional resource identity support. See [HC-FRAMEWORK], [HC-RESOURCES], [HC-IMPORT], [HC-ACTIONS-IMPL], [HC-ACTIONS-INVOKE], [HC-ACCTEST], [HC-DOCS], [HC-IDENTITY], [HC-SCAFFOLD].

---
