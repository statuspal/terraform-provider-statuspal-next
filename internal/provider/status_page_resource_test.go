// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// statusPageMux returns a stateful mock of the status-page endpoints. POST/PATCH
// merge the request body onto the stored resource and GET echoes it back, so a
// post-apply refresh sees exactly what was written (no spurious diffs).
func statusPageMux(subdomain string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":                     "sp_01hxyz",
		"current_status":         "operational",
		"require_authentication": false,
		"notification_settings": map[string]any{
			"email_enabled":    true,
			"slack_enabled":    false,
			"rss_feed_enabled": true,
		},
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		writeData(w, http.StatusCreated, current)
	})
	mux.HandleFunc("/status_pages/"+subdomain, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPatch:
			merge(current, decodeBody(r))
			writeData(w, http.StatusOK, current)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeData(w, http.StatusOK, current)
		}
	})
	return mux
}

func TestAccStatusPageResource(t *testing.T) {
	server := httptest.NewServer(statusPageMux("tf-acc-test-sp"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read.
			{
				Config: cfg + `
resource "statuspal-next_status_page" "test" {
  name        = "TF Acc Test"
  subdomain   = "tf-acc-test-sp"
  timezone    = "America/New_York"
  website_url = "https://tf-acc.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "name", "TF Acc Test"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "subdomain", "tf-acc-test-sp"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "timezone", "America/New_York"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "require_authentication", "false"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "current_status", "operational"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "notification_settings.email_enabled", "true"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "notification_settings.slack_enabled", "false"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "notification_settings.rss_feed_enabled", "true"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "id", "sp_01hxyz"),
					resource.TestCheckResourceAttrSet("statuspal-next_status_page.test", "created_at"),
				),
			},
			// Import by subdomain.
			{
				ResourceName:      "statuspal-next_status_page.test",
				ImportState:       true,
				ImportStateId:     "tf-acc-test-sp",
				ImportStateVerify: true,
			},
			// Update.
			{
				Config: cfg + `
resource "statuspal-next_status_page" "test" {
  name        = "TF Acc Test Renamed"
  subdomain   = "tf-acc-test-sp"
  timezone    = "America/New_York"
  website_url = "https://tf-acc.example.com"

  require_authentication = true

  notification_settings = {
    email_enabled    = false
    slack_enabled    = true
    rss_feed_enabled = true
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "name", "TF Acc Test Renamed"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "require_authentication", "true"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "notification_settings.email_enabled", "false"),
					resource.TestCheckResourceAttr("statuspal-next_status_page.test", "notification_settings.slack_enabled", "true"),
				),
			},
			// Delete happens automatically at the end of the test case.
		},
	})
}
