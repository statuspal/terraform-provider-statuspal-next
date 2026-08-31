// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// monitoringCheckMux is a stateful mock of the monitoring checks API. It derives
// check_type from the url scheme and applies the server-side defaults the
// provider relies on, mirroring the real endpoint.
func monitoringCheckMux(id string) http.Handler {
	var mu sync.Mutex
	current := map[string]any{
		"id":                          id,
		"recv_timeout_secs":           float64(1),
		"geo_areas":                   []any{},
		"display_response_time_chart": false,
		"status":                      nil,
		"automation":                  nil,
		"created_at":                  "2026-01-01T00:00:00Z",
		"updated_at":                  "2026-01-01T00:00:00Z",
	}

	normalize := func() {
		url, _ := current["url"].(string)
		if strings.HasPrefix(strings.ToLower(url), "tcp://") {
			current["check_type"] = "tcp"
			current["http_method"] = nil
		} else {
			current["check_type"] = "http"
			if m, ok := current["http_method"].(string); !ok || m == "" {
				current["http_method"] = "get"
			}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/monitoring_checks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		merge(current, decodeBody(r))
		current["id"] = id
		normalize()
		writeData(w, http.StatusCreated, current)
	})
	mux.HandleFunc("/monitoring_checks/"+id, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPatch:
			merge(current, decodeBody(r))
			normalize()
			writeData(w, http.StatusOK, current)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeData(w, http.StatusOK, current)
		}
	})
	return mux
}

func TestAccMonitoringCheckResource(t *testing.T) {
	server := httptest.NewServer(monitoringCheckMux("mck_01hxyz"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	const name = "statuspal-next_monitoring_check.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create an HTTP check with status-page automation linked.
				Config: cfg + `
resource "statuspal-next_monitoring_check" "test" {
  name              = "API health"
  url               = "https://api.acme.com"
  http_method       = "head"
  recv_timeout_secs = 5
  geo_areas         = ["US"]
  recipient_emails  = ["ops@acme.com"]

  automation = {
    status_page_subdomain = "acme"
    container_slug        = "eu"
    service_slug          = "api"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "API health"),
					resource.TestCheckResourceAttr(name, "url", "https://api.acme.com"),
					resource.TestCheckResourceAttr(name, "check_type", "http"),
					resource.TestCheckResourceAttr(name, "http_method", "head"),
					resource.TestCheckResourceAttr(name, "recv_timeout_secs", "5"),
					resource.TestCheckResourceAttr(name, "geo_areas.#", "1"),
					resource.TestCheckTypeSetElemAttr(name, "recipient_emails.*", "ops@acme.com"),
					resource.TestCheckResourceAttr(name, "automation.service_slug", "api"),
					resource.TestCheckResourceAttr(name, "automation.container_slug", "eu"),
					resource.TestCheckResourceAttrSet(name, "id"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     "mck_01hxyz",
				ImportStateVerify: true,
			},
			{
				// Update: rename, bump timeout, and drop automation (disables it).
				Config: cfg + `
resource "statuspal-next_monitoring_check" "test" {
  name              = "API health (v2)"
  url               = "https://api.acme.com"
  recv_timeout_secs = 10
  recipient_emails  = ["ops@acme.com", "sre@acme.com"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "API health (v2)"),
					resource.TestCheckResourceAttr(name, "recv_timeout_secs", "10"),
					resource.TestCheckResourceAttr(name, "recipient_emails.#", "2"),
					resource.TestCheckNoResourceAttr(name, "automation.service_slug"),
				),
			},
		},
	})
}

func TestAccMonitoringCheckResource_httpMethodOnTCP(t *testing.T) {
	server := httptest.NewServer(monitoringCheckMux("mck_01hxyz"))
	defer server.Close()
	cfg := providerConfig(server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "statuspal-next_monitoring_check" "test" {
  name             = "DB port"
  url              = "tcp://db.acme.com:5432"
  http_method      = "get"
  recipient_emails = ["ops@acme.com"]
}
`,
				ExpectError: regexp.MustCompile(`http_method cannot be set for a TCP check`),
			},
		},
	})
}
