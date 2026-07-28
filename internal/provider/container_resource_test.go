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

func containerMux(subdomain, slug string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":   "ct_01hxyz",
		"slug": slug,
		"services": []any{
			map[string]any{"slug": "api-gateway", "name": "API Gateway", "status": "ok"},
		},
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/"+subdomain+"/containers", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		current["slug"] = slug
		writeData(w, http.StatusCreated, current)
	})
	mux.HandleFunc("/status_pages/"+subdomain+"/containers/"+slug, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPatch:
			merge(current, decodeBody(r))
			current["slug"] = slug
			writeData(w, http.StatusOK, current)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeData(w, http.StatusOK, current)
		}
	})
	return mux
}

func TestAccContainerResource(t *testing.T) {
	server := httptest.NewServer(containerMux("tf-acc-sp", "eu-region"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "statuspal-next_container" "test" {
  status_page_subdomain = "tf-acc-sp"
  name                  = "EU Region"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_container.test", "status_page_subdomain", "tf-acc-sp"),
					resource.TestCheckResourceAttr("statuspal-next_container.test", "name", "EU Region"),
					resource.TestCheckResourceAttr("statuspal-next_container.test", "slug", "eu-region"),
					resource.TestCheckResourceAttr("statuspal-next_container.test", "services.#", "1"),
					resource.TestCheckResourceAttr("statuspal-next_container.test", "services.0.slug", "api-gateway"),
					resource.TestCheckResourceAttr("statuspal-next_container.test", "services.0.status", "ok"),
					resource.TestCheckResourceAttrSet("statuspal-next_container.test", "id"),
				),
			},
			{
				ResourceName:      "statuspal-next_container.test",
				ImportState:       true,
				ImportStateId:     "tf-acc-sp/eu-region",
				ImportStateVerify: true,
			},
			{
				Config: cfg + `
resource "statuspal-next_container" "test" {
  status_page_subdomain = "tf-acc-sp"
  name                  = "EU Region (Frankfurt)"
}
`,
				Check: resource.TestCheckResourceAttr("statuspal-next_container.test", "name", "EU Region (Frankfurt)"),
			},
		},
	})
}
