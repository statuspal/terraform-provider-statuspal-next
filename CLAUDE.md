# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

A Terraform provider for **StatusPal Next** (codename *spage*), built on the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
It manages status pages, services, containers (regions), and outgoing webhooks as code via the
StatusPal Next [Management API](https://next.statuspal.io/api/v1). This is distinct from the
legacy `statuspal/statuspal` (StatusHQ) provider.

Provider address: `registry.terraform.io/statuspal/statuspal-next`; resources/data sources are
prefixed `statuspal-next_`.

## Commands

```bash
go build ./...                       # compile
make test                            # unit tests (go test ./...)
go test ./internal/client/...        # client unit tests only (no backend)
make testacc                         # acceptance tests (sets TF_ACC=1, mock-driven — no live API)
go test -run TestAccService ./internal/provider/   # single acceptance test (needs TF_ACC=1)
make docs                            # regenerate docs/ via tfplugindocs (go generate ./...)
golangci-lint run                    # lint (config in .golangci.yml)
```

Toolchain pins: Go 1.22.9, Terraform 1.9.8 (`.tool-versions`). CI also matrix-tests Terraform 1.11.

CI (`.github/workflows/test.yml`) runs: build + golangci-lint → `go generate` diff check (docs
must be committed) → client unit tests → acceptance tests across Terraform versions.

## Architecture

Two packages, with a hard boundary between them:

### `internal/client` — HTTP client (package `statuspalnext`)

A thin, typed client for the Management API. Knows nothing about Terraform.

- `client.go` — `Client`, `NewClient`, and the generic plumbing. All requests go through
  `doRequest`, which adds the `Bearer` token, enforces a 10 req/s rate limiter (`golang.org/x/time/rate`),
  and converts any non-2xx into an `*APIError`. Generic helpers `getInto[T]`, `writeInto[T]`,
  `listAll[T]`, and `decodeData[T]` handle the API's response envelopes.
- The API wraps single resources as `{ "data": {...} }` and collections as
  `{ "data": [...], "meta": {...} }`. `listAll` auto-paginates by following `meta.total_pages`
  (a `total_pages` of 0 means the endpoint is non-paginated).
- `models.go` — Go structs mirroring `spage`'s `docs/openapi.yaml`. **Convention:** optional
  *request* fields are pointers (`*bool`, `*string`, `*int64`) so a nil pointer is omitted from a
  PATCH while a non-nil pointer is always sent (even zero values like `false`); read-only
  *response* fields are plain values with `omitempty`. Use the `BoolPtr`/`StringPtr`/`Int64Ptr`
  helpers to build request bodies.
- `errors.go` — `APIError` (parses the `{error, message, details}` envelope, falls back to raw
  body) and `IsNotFound(err)`, which resources use in `Read`/`Delete` to detect out-of-band
  deletion.
- One file per resource (`status_pages.go`, `services.go`, `containers.go`,
  `outgoing_webhooks.go`) exposing typed CRUD methods. Resources nested under a status page take
  a `subdomain` argument; path builders (`statusPagePath`, `servicesPath`) `url.PathEscape` all
  segments.

### `internal/provider` — Terraform layer (package `provider`)

- `provider.go` — `statuspalNextProvider`. `Configure` resolves `api_key`/`endpoint` (explicit
  config overrides the `STATUSPAL_NEXT_API_KEY`/`STATUSPAL_NEXT_ENDPOINT` env vars), builds the
  client, and stashes it in `resp.ResourceData`/`resp.DataSourceData`. **Register every new
  resource/data source** in the `Resources`/`DataSources` slices here.
- `helpers.go` — `clientFromProviderData` (the standard Configure type-assertion every
  resource/data source uses; returns nil without error when providerData is nil, since Terraform
  calls Configure with nil before the provider is configured) and small pointer-deref helpers.
- Each managed type has a `_resource.go` (and where applicable a `_data_source.go`). They follow
  the Plugin Framework idiom: a `tfsdk`-tagged model struct, `Metadata`/`Schema`/`Configure`, and
  `Create`/`Read`/`Update`/`Delete`/`ImportState`. Two helper functions per resource bridge to the
  client: `xRequestBody(plan)` builds the client struct from the plan (only sending set fields),
  and `mapXToModel(...)` maps an API response back into the tfsdk model.

### Conventions when adding a resource/field

- Resources keyed by a parent (services, containers, webhooks scoping) use a composite import ID
  like `"<status_page_subdomain>/<slug>"`, split in `ImportState`.
- Identity attributes that the server owns (`slug`, `id`) use
  `stringplanmodifier.UseStateForUnknown()`; attributes whose change requires recreation (e.g.
  `status_page_subdomain`) use `RequiresReplace()`.
- All source files carry the `Copyright (c) HashiCorp, Inc.` / `SPDX-License-Identifier: MPL-2.0`
  header (enforced by `.copywrite.hcl`).
- After any schema/description change, run `make docs` and commit the regenerated `docs/` — CI
  fails if generated docs differ. Provider descriptions are authored as `MarkdownDescription` on
  schema attributes; `examples/` holds the `.tf` snippets tfplugindocs embeds.

### Testing

Acceptance tests are **mock-driven** (like the StatusPal Classic provider): each test stands up an
`httptest.Server`, points the provider at it via `endpoint`, and exercises the full Terraform
lifecycle. They set `TF_ACC=1` but need no live backend or secrets. Shared mock helpers
(`providerConfig`, `writeData`, `writeList`, `decodeBody`, `merge`) live in `provider_test.go`.
Client-level unit tests in `internal/client` need neither `TF_ACC` nor a backend.

## Monitoring checks & incident automation

Both are now supported (the Management API exposes them):

- `statuspal-next_monitoring_check` — org-scoped, TypeID-keyed (like `outgoing_webhook`), at the
  top-level `/monitoring_checks` endpoint. Uses human-friendly inputs (`url` with the scheme
  selecting http/tcp; `recipient_emails` resolved to org users server-side). Its optional
  `automation` block (`status_page_subdomain`/`container_slug`/`service_slug`) links the check to a
  status-page service for incident automation; it is sent without `omitempty` so an absent block
  serializes as `null` and disables automation.
- `statuspal-next_automation` — status-page-scoped (nested under `/status_pages/{subdomain}/
  automations`), TypeID-keyed, composite import ID `"<subdomain>/<id>"`. The provider exposes only
  **inline custom** JSONPath formats (`automation_format`); the API's named shared formats are not
  surfaced yet (see README Roadmap). `secret` is write-only (preserved from state on read, like the
  webhook signing secret); `trigger_url` is the computed POST target.

## Not yet supported

- Named (shared) automation formats for `statuspal-next_automation` (inline custom formats only).
- Outgoing webhook secret rotation.
