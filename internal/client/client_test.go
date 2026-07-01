// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := NewClient(StringPtr("sk_test_token"), &srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient(nil, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.HostURL != DefaultEndpoint {
		t.Errorf("HostURL = %q, want %q", c.HostURL, DefaultEndpoint)
	}
	if c.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", c.APIKey)
	}

	// A trailing slash on the endpoint must be trimmed so path joins stay clean.
	c, err = NewClient(StringPtr("k"), StringPtr("https://example.test/api/v1/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.HostURL != "https://example.test/api/v1" {
		t.Errorf("HostURL = %q, want trimmed", c.HostURL)
	}
}

func TestRequestHeadersAndEnvelope(t *testing.T) {
	var gotAuth, gotAccept, gotPath, gotMethod string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":{"id":"sp_1","name":"Acme","subdomain":"acme"}}`)
	})

	sp, err := c.GetStatusPage(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetStatusPage: %v", err)
	}
	if gotAuth != "Bearer sk_test_token" {
		t.Errorf("Authorization = %q, want Bearer token", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
	if gotMethod != http.MethodGet || gotPath != "/status_pages/acme" {
		t.Errorf("got %s %s, want GET /status_pages/acme", gotMethod, gotPath)
	}
	if sp.ID != "sp_1" || sp.Name != "Acme" || sp.Subdomain != "acme" {
		t.Errorf("decoded status page = %+v", sp)
	}
}

func TestCreateStatusPageSendsBody(t *testing.T) {
	var body map[string]any
	var method string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"id":"sp_1","name":"Acme","subdomain":"acme"}}`)
	})

	_, err := c.CreateStatusPage(context.Background(), &StatusPage{
		Name:                  "Acme",
		Subdomain:             "acme",
		Timezone:              "UTC",
		WebsiteURL:            "https://acme.test",
		RequireAuthentication: BoolPtr(false),
		NotificationSettings:  &NotificationSettings{EmailEnabled: BoolPtr(true)},
	})
	if err != nil {
		t.Fatalf("CreateStatusPage: %v", err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %s, want POST", method)
	}
	if body["subdomain"] != "acme" || body["website_url"] != "https://acme.test" {
		t.Errorf("request body = %+v", body)
	}
	// require_authentication must be serialized even when false (pointer field).
	if _, ok := body["require_authentication"]; !ok {
		t.Errorf("require_authentication missing from body: %+v", body)
	}
}

func TestUpdateUsesPatch(t *testing.T) {
	var method string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":{"id":"sp_1","subdomain":"acme"}}`)
	})

	_, err := c.UpdateStatusPage(context.Background(), "acme", &StatusPage{Name: "New"})
	if err != nil {
		t.Fatalf("UpdateStatusPage: %v", err)
	}
	if method != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", method)
	}
}

func TestDeleteTreats204AsSuccess(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent) // empty body
	})

	if err := c.DeleteStatusPage(context.Background(), "acme"); err != nil {
		t.Fatalf("DeleteStatusPage: %v", err)
	}
}

func TestErrorParsing(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"error":"unprocessable_entity","message":"Validation failed","details":{"subdomain":["has already been taken"]}}`)
	})

	_, err := c.GetStatusPage(context.Background(), "acme")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"422", "unprocessable_entity", "Validation failed", "subdomain", "has already been taken"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q missing %q", msg, want)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"not_found","message":"The requested resource was not found"}`)
	})

	_, err := c.GetService(context.Background(), "acme", "missing")
	if !IsNotFound(err) {
		t.Errorf("IsNotFound = false for 404 error: %v", err)
	}
}

func TestListPaginates(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.WriteHeader(http.StatusOK)
		switch page {
		case "1":
			_, _ = fmt.Fprint(w, `{"data":[{"id":"sp_1","subdomain":"a"}],"meta":{"current_page":1,"total_pages":2}}`)
		case "2":
			_, _ = fmt.Fprint(w, `{"data":[{"id":"sp_2","subdomain":"b"}],"meta":{"current_page":2,"total_pages":2}}`)
		default:
			t.Errorf("unexpected page %q", page)
		}
	})

	pages, err := c.ListStatusPages(context.Background())
	if err != nil {
		t.Fatalf("ListStatusPages: %v", err)
	}
	if len(pages) != 2 || pages[0].Subdomain != "a" || pages[1].Subdomain != "b" {
		t.Errorf("pages = %+v, want 2 across both pages", pages)
	}
}
