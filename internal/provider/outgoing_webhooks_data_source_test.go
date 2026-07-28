// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOutgoingWebhooksDataSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/outgoing_webhooks", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, []any{
			map[string]any{
				"id":                     "wbk_1",
				"name":                   "Ops alerts",
				"url":                    "https://hooks.example.com/statuspal",
				"events":                 []any{"notice.created", "notice.updated"},
				"enabled":                true,
				"all_status_pages":       false,
				"status_page_subdomains": []any{"prod-status"},
				"created_at":             "2026-01-01T00:00:00Z",
				"updated_at":             "2026-01-01T00:00:00Z",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(server.URL) + `data "statuspal-next_outgoing_webhooks" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.statuspal-next_outgoing_webhooks.all", "outgoing_webhooks.#", "1"),
					resource.TestCheckResourceAttr("data.statuspal-next_outgoing_webhooks.all", "outgoing_webhooks.0.name", "Ops alerts"),
					resource.TestCheckResourceAttr("data.statuspal-next_outgoing_webhooks.all", "outgoing_webhooks.0.all_status_pages", "false"),
					resource.TestCheckResourceAttr("data.statuspal-next_outgoing_webhooks.all", "outgoing_webhooks.0.events.#", "2"),
					resource.TestCheckResourceAttr("data.statuspal-next_outgoing_webhooks.all", "outgoing_webhooks.0.status_page_subdomains.#", "1"),
				),
			},
		},
	})
}
