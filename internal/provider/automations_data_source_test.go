// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAutomationsDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/acme/automations", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, []any{
			map[string]any{
				"id":               "auto_1",
				"service_slug":     "api",
				"container_slug":   "eu",
				"manage_incidents": true,
				"has_secret":       true,
				"automation_format": map[string]any{
					"name":                 "Custom - api - eu",
					"expected_result_path": "$.status",
					"expected_result":      "ok",
					"secret_path":          "$.token",
					"custom":               true,
				},
				"trigger_url": "http://spage.test/incident_automations/tok_1/trigger",
				"created_at":  "2026-01-01T00:00:00Z",
				"updated_at":  "2026-01-01T00:00:00Z",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	const ds = "data.statuspal-next_automations.acme"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `data "statuspal-next_automations" "acme" { status_page_subdomain = "acme" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ds, "automations.#", "1"),
					resource.TestCheckResourceAttr(ds, "automations.0.service_slug", "api"),
					resource.TestCheckResourceAttr(ds, "automations.0.manage_incidents", "true"),
					resource.TestCheckResourceAttr(ds, "automations.0.has_secret", "true"),
					resource.TestCheckResourceAttr(ds, "automations.0.expected_result_path", "$.status"),
					resource.TestCheckResourceAttrSet(ds, "automations.0.trigger_url"),
				),
			},
		},
	})
}
