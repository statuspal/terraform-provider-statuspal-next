// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStatusPageDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/acme-status", func(w http.ResponseWriter, r *http.Request) {
		writeData(w, http.StatusOK, sampleStatusPage("acme-status", "Acme Status"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `
data "statuspal-next_status_page" "acme" {
  subdomain = "acme-status"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.statuspal-next_status_page.acme", "subdomain", "acme-status"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_page.acme", "name", "Acme Status"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_page.acme", "timezone", "America/New_York"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_page.acme", "current_status", "operational"),
					resource.TestCheckResourceAttr("data.statuspal-next_status_page.acme", "notification_settings.rss_feed_enabled", "true"),
				),
			},
		},
	})
}
