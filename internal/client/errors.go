// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// APIError represents a non-2xx response from the Management API. The API
// returns errors as `{ "error": "...", "message": "...", "details": { ... } }`;
// any field may be absent, so Raw retains the original body as a fallback.
type APIError struct {
	StatusCode int                 `json:"-"`
	Code       string              `json:"error"`
	Message    string              `json:"message"`
	Details    map[string][]string `json:"details"`
	Raw        string              `json:"-"`
}

// NewAPIError parses the error envelope out of an HTTP error response body.
func NewAPIError(status int, body []byte) error {
	e := &APIError{StatusCode: status, Raw: strings.TrimSpace(string(body))}
	// Best-effort decode; an unparseable body still yields a useful error via Raw.
	_ = json.Unmarshal(body, e)
	return e
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "status %d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " (%s)", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}

	if len(e.Details) > 0 {
		keys := make([]string, 0, len(e.Details))
		for k := range e.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, strings.Join(e.Details[k], ", ")))
		}
		fmt.Fprintf(&b, " [%s]", strings.Join(parts, "; "))
	}

	// If the structured fields were empty but we have a raw body, surface it so
	// the error is never just "status 500".
	if e.Code == "" && e.Message == "" && len(e.Details) == 0 && e.Raw != "" {
		fmt.Fprintf(&b, ": %s", e.Raw)
	}

	return b.String()
}

// IsNotFound reports whether err is an APIError with a 404 status. Resources use
// this in Read to drop themselves from state when deleted out-of-band.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
