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

// automationMux mocks the nested automations API for a single status page. It
// echoes the custom format back and derives a trigger_url, like spage does.
func automationMux(subdomain, id string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":               id,
		"manage_incidents": false,
		"created_at":       "2026-01-01T00:00:00Z",
		"updated_at":       "2026-01-01T00:00:00Z",
	}

	render := func() map[string]any {
		out := cloneMap(current)
		out["id"] = id
		out["trigger_url"] = "http://spage.test/incident_automations/tok_" + id + "/trigger"
		out["has_secret"] = current["secret"] != nil && current["secret"] != ""
		delete(out, "secret")
		delete(out, "automation_format_name")
		return out
	}

	base := "/status_pages/" + subdomain + "/automations"
	mux := http.NewServeMux()
	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		// Mark inline formats custom, as the server does.
		if f, ok := current["automation_format"].(map[string]any); ok {
			f["custom"] = true
			f["name"] = "Custom - api - eu"
		}
		writeData(w, http.StatusCreated, render())
	})
	mux.HandleFunc(base+"/"+id, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPatch:
			merge(current, decodeBody(r))
			writeData(w, http.StatusOK, render())
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeData(w, http.StatusOK, render())
		}
	})
	return mux
}

func TestAccAutomationResource(t *testing.T) {
	server := httptest.NewServer(automationMux("acme", "auto_01hxyz"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	const name = "statuspal-next_automation.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "statuspal-next_automation" "test" {
  status_page_subdomain = "acme"
  service_slug          = "api"
  container_slug        = "eu"
  manage_incidents      = true
  secret                = "s3cr3t"

  automation_format = {
    expected_result_path = "$.status"
    expected_result      = "ok"
    secret_path          = "$.token"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "service_slug", "api"),
					resource.TestCheckResourceAttr(name, "container_slug", "eu"),
					resource.TestCheckResourceAttr(name, "manage_incidents", "true"),
					resource.TestCheckResourceAttr(name, "automation_format.expected_result_path", "$.status"),
					resource.TestCheckResourceAttr(name, "automation_format.expected_result", "ok"),
					resource.TestCheckResourceAttr(name, "automation_format.secret_path", "$.token"),
					resource.TestCheckResourceAttr(name, "secret", "s3cr3t"),
					resource.TestCheckResourceAttrSet(name, "trigger_url"),
					resource.TestCheckResourceAttrSet(name, "id"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateId:           "acme/auto_01hxyz",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{
				Config: cfg + `
resource "statuspal-next_automation" "test" {
  status_page_subdomain = "acme"
  service_slug          = "api"
  container_slug        = "eu"
  manage_incidents      = false

  automation_format = {
    expected_result_path = "$.health.state"
    expected_result      = "healthy"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "manage_incidents", "false"),
					resource.TestCheckResourceAttr(name, "automation_format.expected_result_path", "$.health.state"),
					resource.TestCheckNoResourceAttr(name, "automation_format.secret_path"),
				),
			},
		},
	})
}
