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

func outgoingWebhookMux(id string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":               id,
		"enabled":          true,
		"all_status_pages": true,
		"created_at":       "2026-01-01T00:00:00Z",
		"updated_at":       "2026-01-01T00:00:00Z",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/outgoing_webhooks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		current["id"] = id
		// The signing secret is returned only on create.
		resp := cloneMap(current)
		resp["secret"] = "whs_secret_xyz"
		writeData(w, http.StatusCreated, resp)
	})
	mux.HandleFunc("/outgoing_webhooks/"+id, func(w http.ResponseWriter, r *http.Request) {
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

func TestAccOutgoingWebhookResource(t *testing.T) {
	server := httptest.NewServer(outgoingWebhookMux("wbk_01hxyz"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "statuspal-next_outgoing_webhook" "test" {
  name                   = "Ops alerts"
  url                    = "https://hooks.example.com/statuspal"
  events                 = ["notice.created", "notice.updated", "service.status_changed"]
  all_status_pages       = false
  status_page_subdomains = ["tf-acc-sp"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "name", "Ops alerts"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "url", "https://hooks.example.com/statuspal"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "enabled", "true"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "all_status_pages", "false"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "events.#", "3"),
					resource.TestCheckTypeSetElemAttr("statuspal-next_outgoing_webhook.test", "events.*", "notice.created"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "status_page_subdomains.#", "1"),
					resource.TestCheckTypeSetElemAttr("statuspal-next_outgoing_webhook.test", "status_page_subdomains.*", "tf-acc-sp"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "secret", "whs_secret_xyz"),
					resource.TestCheckResourceAttrSet("statuspal-next_outgoing_webhook.test", "id"),
				),
			},
			{
				ResourceName:      "statuspal-next_outgoing_webhook.test",
				ImportState:       true,
				ImportStateId:     "wbk_01hxyz",
				ImportStateVerify: true,
				// The signing secret cannot be recovered on import (only returned at create).
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{
				Config: cfg + `
resource "statuspal-next_outgoing_webhook" "test" {
  name                   = "Ops alerts (paused)"
  url                    = "https://hooks.example.com/statuspal"
  events                 = ["notice.created"]
  enabled                = false
  all_status_pages       = false
  status_page_subdomains = ["tf-acc-sp"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "name", "Ops alerts (paused)"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "enabled", "false"),
					resource.TestCheckResourceAttr("statuspal-next_outgoing_webhook.test", "events.#", "1"),
				),
			},
		},
	})
}
