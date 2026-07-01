// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServicesDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/acme-status/services", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, []any{
			map[string]any{
				"id":          "svc_1",
				"slug":        "api-gateway",
				"name":        "API Gateway",
				"description": "Public REST API",
				"order":       1,
				"status":      "ok",
				"container_statuses": []any{
					map[string]any{"container_slug": "default", "status": "ok"},
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
data "statuspal-next_services" "acme" {
  status_page_subdomain = "acme-status"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.statuspal-next_services.acme", "services.#", "1"),
					resource.TestCheckResourceAttr("data.statuspal-next_services.acme", "services.0.slug", "api-gateway"),
					resource.TestCheckResourceAttr("data.statuspal-next_services.acme", "services.0.name", "API Gateway"),
					resource.TestCheckResourceAttr("data.statuspal-next_services.acme", "services.0.status", "ok"),
					resource.TestCheckResourceAttr("data.statuspal-next_services.acme", "services.0.container_statuses.0.container_slug", "default"),
				),
			},
		},
	})
}
