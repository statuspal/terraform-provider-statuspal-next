// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func sampleStatusPage(subdomain, name string) map[string]any {
	return map[string]any{
		"id":                     "sp_" + subdomain,
		"name":                   name,
		"subdomain":              subdomain,
		"timezone":               "America/New_York",
		"website_url":            "https://" + subdomain + ".example.com",
		"require_authentication": false,
		"current_status":         "operational",
		"notification_settings": map[string]any{
			"email_enabled":    true,
			"slack_enabled":    false,
			"rss_feed_enabled": true,
		},
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
	}
}

func TestAccStatusPagesDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, []any{sampleStatusPage("acme-status", "Acme Status")})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `data "statuspal-next_status_pages" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.statuspal-next_status_pages.all", "status_pages.#", "1"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_pages.all", "status_pages.0.subdomain", "acme-status"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_pages.all", "status_pages.0.name", "Acme Status"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_pages.all", "status_pages.0.current_status", "operational"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_pages.all", "status_pages.0.notification_settings.email_enabled", "true"),
				),
			},
		},
	})
}
