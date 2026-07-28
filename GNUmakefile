default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ENV=TEST TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Run unit tests
.PHONY: test
test:
	go test ./... $(TESTARGS)

# Generate documentation
.PHONY: docs
docs:
	go generate ./...

# End-to-end test against a real backend. Builds + installs the provider locally
# (resolved via the dev_overrides block in ~/.terraformrc — see e2e-test/README.md)
# and runs the full lifecycle. Credentials come from STATUSPAL_NEXT_API_KEY /
# STATUSPAL_NEXT_ENDPOINT or e2e-test/terraform.tfvars.
.PHONY: e2e
e2e:
	go install .
	terraform -chdir=e2e-test apply $(TESTARGS)

.PHONY: e2e-destroy
e2e-destroy:
	terraform -chdir=e2e-test destroy $(TESTARGS)
