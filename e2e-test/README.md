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
is shown once).

## 4. Run the lifecycle

```bash
terraform -chdir=e2e-test plan
terraform -chdir=e2e-test apply  -auto-approve
terraform -chdir=e2e-test destroy -auto-approve
```

A clean run is: apply succeeds, `service_count` output is ≥ 1, re-running `plan`
shows **no changes** (no perpetual diff), and `destroy` removes everything.
