// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate the provider during
// acceptance testing. The factory is invoked for every Terraform CLI command so
// the CLI can reattach to the in-process provider server.
//
// Like the StatusPal Classic provider, the acceptance tests here are
// mock-driven: each test stands up an httptest server, points the provider at
// it via the `endpoint` argument, and exercises the full Terraform lifecycle.
// They require TF_ACC=1 (enforced by the testing framework) but need no live
// StatusPal Next backend.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"statuspal-next": providerserver.NewProtocol6WithError(New("test")()),
}

// providerConfig returns a provider block pointing at the given mock endpoint.
// Prepend it to each test step's configuration.
func providerConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "statuspal-next" {
  api_key  = "sk_test"
  endpoint = %q
}
`, endpoint)
}

// decodeBody decodes a JSON request body into a generic map (empty on no body).
func decodeBody(r *http.Request) map[string]any {
	m := map[string]any{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&m)
	}
	return m
}

// merge shallow-copies src over dst. Nested objects are replaced wholesale,
// which matches how the API echoes back the fields it was sent.
func merge(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

// cloneMap returns a shallow copy of m.
func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// writeData writes a single-resource `{ "data": ... }` envelope.
func writeData(w http.ResponseWriter, status int, obj any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": obj})
}

// writeList writes a collection `{ "data": [...], "meta": {...} }` envelope.
func writeList(w http.ResponseWriter, items []any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
		"meta": map[string]any{
			"current_page": 1,
			"per_page":     100,
			"total_count":  len(items),
			"total_pages":  1,
		},
	})
}
