// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMonitoringChecksDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/monitoring_checks", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, []any{
			map[string]any{
				"id":                          "mck_1",
				"name":                        "API health",
				"url":                         "https://api.acme.com",
				"check_type":                  "http",
				"http_method":                 "get",
				"recv_timeout_secs":           5,
				"geo_areas":                   []any{"US"},
				"recipient_emails":            []any{"ops@acme.com"},
				"display_response_time_chart": false,
				"status":                      "up",
				"automation": map[string]any{
					"status_page_subdomain": "acme",
					"container_slug":        "eu",
					"service_slug":          "api",
				},
				"created_at": "2026-01-01T00:00:00Z",
				"updated_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	const ds = "data.statuspal-next_monitoring_checks.all"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `data "statuspal-next_monitoring_checks" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ds, "monitoring_checks.#", "1"),
					resource.TestCheckResourceAttr(ds, "monitoring_checks.0.name", "API health"),
					resource.TestCheckResourceAttr(ds, "monitoring_checks.0.check_type", "http"),
					resource.TestCheckResourceAttr(ds, "monitoring_checks.0.status", "up"),
					resource.TestCheckResourceAttr(ds, "monitoring_checks.0.automation.service_slug", "api"),
				),
			},
		},
	})
}
