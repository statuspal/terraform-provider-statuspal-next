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

func serviceMux(subdomain, slug string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":     "svc_01hxyz",
		"slug":   slug,
		"status": "ok",
		"order":  1,
		"container_statuses": []any{
			map[string]any{"container_slug": "default", "status": "ok"},
		},
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status_pages/"+subdomain+"/services", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		current["slug"] = slug // server controls the slug
		writeData(w, http.StatusCreated, current)
	})
	mux.HandleFunc("/status_pages/"+subdomain+"/services/"+slug, func(w http.ResponseWriter, r *http.Request) {
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

func TestAccServiceResource(t *testing.T) {
	server := httptest.NewServer(serviceMux("tf-acc-sp", "api-gateway"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "statuspal-next_service" "test" {
  status_page_subdomain = "tf-acc-sp"
  name                  = "API Gateway"
  description           = "Public REST API"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("statuspal-next_service.test", "status_page_subdomain", "tf-acc-sp"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "name", "API Gateway"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "slug", "api-gateway"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "description", "Public REST API"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "status", "ok"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "order", "1"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "container_statuses.#", "1"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "container_statuses.0.container_slug", "default"),
					resource.TestCheckResourceAttr("statuspal-next_service.test", "container_statuses.0.status", "ok"),
					resource.TestCheckResourceAttrSet("statuspal-next_service.test", "id"),
				),
			},
			{
				ResourceName:      "statuspal-next_service.test",
				ImportState:       true,
				ImportStateId:     "tf-acc-sp/api-gateway",
				ImportStateVerify: true,
			},
			{
				Config: cfg + `
resource "statuspal-next_service" "test" {
  status_page_subdomain = "tf-acc-sp"
  name                  = "API Gateway v2"
  description           = "Public REST API"
}
`,
				Check: resource.TestCheckResourceAttr("statuspal-next_service.test", "name", "API Gateway v2"),
			},
		},
	})
}
