# End-to-end test

Exercises the full resource set (`status_page`, `container`, `service`,
`outgoing_webhook`, `monitoring_check`, `automation` + the `services`,
`monitoring_checks`, and `automations` data sources) against a **real** StatusPal Next
backend, using a locally-built provider via Terraform `dev_overrides`. This mirrors
the `e2e-test/` harness in the classic `statuspal/statuspal` provider.

> Creates and destroys real resources. Point it at a test org, not production data
> you care about. `terraform.tfvars` and state files are git-ignored.

## 1. Build & install the provider locally

```bash
go install .            # from the repo root
```

`go install` puts the binary in `$GOBIN` if set, otherwise `$GOPATH/bin`. Find the
exact directory to use in the next step with:

```bash
go env GOBIN     # if this prints a path, use it; if empty, use:
echo "$(go env GOPATH)/bin"
```

## 2. Add a dev_overrides block to `~/.terraformrc`

Use the directory from step 1 (the bin dir, not the binary itself):

```hcl
provider_installation {
  dev_overrides {
    "statuspal/statuspal-next" = "/Users/you/go/bin"
  }
  direct {}
}
```

With `dev_overrides` active, **skip `terraform init`** — Terraform uses the local
binary directly and will print a warning saying so (that's expected).

## 3. Provide credentials

Either copy `terraform.tfvars.example` → `terraform.tfvars` and fill it in, or export:

```bash
export STATUSPAL_NEXT_API_KEY=sk_...
export STATUSPAL_NEXT_ENDPOINT=https://next.statuspal.io/api/v1   # or a local instance
```

Create an org API key from the StatusPal Next UI if you don't have one (the `sk_…` token
is shown once). See the root README for the Rails-console alternative.

### Monitoring checks need Sentinel (local gotcha)

Creating/updating/deleting a `monitoring_check` makes the backend call the external
Sentinel monitoring service. spage reads `SENTINEL_HOST`/`SENTINEL_API_KEY` from its env
(dev `.env` defaults to `http://localhost:4080`). If nothing is listening there, the API
returns `502 monitoring_service_error` and the apply fails. Pick one:

- **Have Sentinel running** at the configured host, **or**
- **Stub it** — point `SENTINEL_HOST` at any server that returns `2xx` for
  `POST /subscription`, `PUT /api/v1/subscriptions/:id`, and `DELETE /subscription/:id`, **or**
- **Disable it** — unset `SENTINEL_HOST`/`SENTINEL_API_KEY` on the spage server so the
  integration becomes a no-op (the check is still created, just not registered for monitoring).

This only affects `monitoring_check` resources; the rest of the harness needs no Sentinel.

## 4. Run the lifecycle

```bash
terraform -chdir=e2e-test plan
terraform -chdir=e2e-test apply  -auto-approve
terraform -chdir=e2e-test destroy -auto-approve
```

A clean run is: apply succeeds, `service_count` output is ≥ 1, re-running `plan`
shows **no changes** (no perpetual diff), and `destroy` removes everything.
