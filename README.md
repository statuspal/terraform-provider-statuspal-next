# Terraform Provider for StatusPal Next

A [Terraform](https://www.terraform.io) provider for [StatusPal Next](https://www.statuspal.io)
(codename *spage*), built on the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
It manages status pages and their building blocks as code through the StatusPal Next
[Management API](https://next.statuspal.io/api/v1).

> This is the provider for **StatusPal Next**. For the legacy StatusHQ product, use the
> [`statuspal/statuspal`](https://registry.terraform.io/providers/statuspal/statuspal) provider.

## Supported resources & data sources

| Type | Resource | Data source |
|---|---|---|
| Status pages | `statuspal-next_status_page` | `statuspal-next_status_page`, `statuspal-next_status_pages` |
| Services | `statuspal-next_service` | `statuspal-next_services` |
| Containers (regions) | `statuspal-next_container` | `statuspal-next_containers` |
| Outgoing webhooks | `statuspal-next_outgoing_webhook` | `statuspal-next_outgoing_webhooks` |
| Monitoring checks | `statuspal-next_monitoring_check` | `statuspal-next_monitoring_checks` |
| Incident automations | `statuspal-next_automation` | `statuspal-next_automations` |

A `statuspal-next_monitoring_check` can drive **status-page incident automation** directly: set
its `automation` block to link the check to a status-page service, and StatusPal opens an incident
when the check goes down and resolves it when it recovers. `statuspal-next_automation` is the
complementary webhook-driven path — an external monitor POSTs to a generated `trigger_url`.

## Usage

```hcl
terraform {
  required_providers {
    statuspal-next = {
      source = "statuspal/statuspal-next"
    }
  }
}

provider "statuspal-next" {
  api_key = var.statuspal_next_api_key # or STATUSPAL_NEXT_API_KEY
}

resource "statuspal-next_status_page" "acme" {
  name        = "Acme Status"
  subdomain   = "acme-status"
  timezone    = "America/New_York"
  website_url = "https://acme.com"
}

resource "statuspal-next_service" "api" {
  status_page_subdomain = statuspal-next_status_page.acme.subdomain
  name                  = "API Gateway"
}
```

### Provider configuration

| Argument | Env var | Default | Description |
|---|---|---|---|
| `api_key` | `STATUSPAL_NEXT_API_KEY` | — | Organization API key (`sk_…`), sent as a Bearer token. Required. |
| `endpoint` | `STATUSPAL_NEXT_ENDPOINT` | `https://next.statuspal.io/api/v1` | Base URL of the Management API. Point at a local instance for development. |

### Obtaining an API key

Create an organization API key from the StatusPal Next UI. The raw token (`sk_…`) is
shown once on creation — copy it then.

Alternatively, from the Rails console:

```ruby
key = ApiKey.create!(organization: Organization.find_by(name: "Your Org"), name: "terraform")
puts key.raw_token # shown once
```

## Development

Requires Go (see `go.mod`) and Terraform.

```bash
go build ./...                  # compile
go test ./internal/client/...   # unit tests (no backend needed)
go generate ./...               # regenerate docs/ via tfplugindocs
golangci-lint run               # lint
```

### Running against a local spage instance

1. Start spage (`bin/dev`, served at `http://spage.test:7070`).
2. Issue and activate an API key (see above).
3. Build the provider and add a `dev_overrides` block to `~/.terraformrc`:

   ```hcl
   provider_installation {
     dev_overrides {
       "statuspal/statuspal-next" = "/path/to/go/bin"
     }
     direct {}
   }
   ```

4. Configure the provider with `endpoint = "http://spage.test:7070/api/v1"` and your key,
   then `terraform plan` / `apply`.

### Acceptance tests

Acceptance tests are **mock-driven**: each test stands up an `httptest` server and
points the provider at it, exercising the full Terraform lifecycle. They set `TF_ACC=1`
but need **no live backend or credentials** (this matches the StatusPal Classic provider).

```bash
make testacc            # or: TF_ACC=1 go test -v ./internal/provider/
```

> If you use `asdf` and the tests fail with `No version is set for command terraform`,
> the plugin framework is running `terraform` from a temp dir with no `.tool-versions`.
> Pin the version: `TF_ACC=1 ASDF_TERRAFORM_VERSION=1.9.8 go test ./internal/provider/`.

### End-to-end tests (real backend)

To exercise the provider against a real StatusPal Next instance — the way the classic
provider's `e2e-test/` works — see [`e2e-test/`](./e2e-test/). It builds the provider
locally (`dev_overrides`) and runs a full `apply`/`destroy` against your org:

```bash
make e2e   # builds + installs the provider, then `terraform -chdir=e2e-test apply`
```

Credentials come from `STATUSPAL_NEXT_API_KEY` / `STATUSPAL_NEXT_ENDPOINT` (or
`e2e-test/terraform.tfvars`). See [`e2e-test/README.md`](./e2e-test/README.md) for the
one-time `~/.terraformrc` setup.

## Roadmap

- Named (shared) automation formats — `statuspal-next_automation` currently exposes only inline
  custom JSONPath formats; the Management API also supports selecting a shared format by name.
- Outgoing webhook secret rotation.

## License

[MPL-2.0](./LICENSE).
