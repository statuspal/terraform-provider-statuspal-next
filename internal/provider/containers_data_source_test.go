// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContainersDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/acme-status/containers", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, []any{
			map[string]any{
				"id":   "ct_1",
				"slug": "default",
				"name": "Default",
				"services": []any{
					map[string]any{"slug": "api-gateway", "name": "API Gateway", "status": "ok"},
				},
				"created_at": "2026-01-01T00:00:00Z",
				"updated_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
data "statuspal-next_containers" "acme" {
  status_page_subdomain = "acme-status"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.statuspal-next_containers.acme", "containers.#", "1"),
					resource.TestCheckResourceAttr("data.statuspal-next_containers.acme", "containers.0.slug", "default"),
					resource.TestCheckResourceAttr("data.statuspal-next_containers.acme", "containers.0.name", "Default"),
					resource.TestCheckResourceAttr("data.statuspal-next_containers.acme", "containers.0.services.0.slug", "api-gateway"),
				),
			},
		},
	})
}
