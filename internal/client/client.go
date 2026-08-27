// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package statuspalnext is a thin HTTP client for the StatusPal Next
// Management API (https://next.statuspal.io/api/v1). It is intentionally small:
// it handles bearer authentication, the `{ "data": ... }` response envelope,
// rate limiting, and decoding of API errors, and exposes typed CRUD helpers for
// each resource the Terraform provider manages.
package statuspalnext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// DefaultEndpoint is the production base URL of the Management API.
const DefaultEndpoint = "https://next.statuspal.io/api/v1"

const (
	// rateLimit is the steady-state requests-per-second the client allows.
	rateLimit = 10
	// burstLimit is the number of requests that may be bursted.
	burstLimit = 10
	// requestTimeout bounds a single HTTP request.
	requestTimeout = 30 * time.Second
)

// Client talks to the StatusPal Next Management API.
type Client struct {
	HostURL    string
	HTTPClient *http.Client
	APIKey     string

	limiter *rate.Limiter
}

// NewClient builds a Client. endpoint defaults to DefaultEndpoint when nil or
// empty; a trailing slash is trimmed so path joins stay clean. apiKey may be
// empty (e.g. during provider validation), in which case no Authorization
// header is sent.
func NewClient(apiKey, endpoint *string) (*Client, error) {
	c := &Client{
		HTTPClient: &http.Client{Timeout: requestTimeout},
		HostURL:    DefaultEndpoint,
		limiter:    rate.NewLimiter(rateLimit, burstLimit),
	}

	if endpoint != nil && *endpoint != "" {
		c.HostURL = strings.TrimRight(*endpoint, "/")
	}

	if apiKey != nil {
		c.APIKey = *apiKey
	}

	return c, nil
}

// dataEnvelope wraps a single-resource response: { "data": { ... } }.
type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

// listEnvelope wraps a collection response: { "data": [ ... ], "meta": { ... } }.
type listEnvelope[T any] struct {
	Data []T            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginationMeta is the pagination block returned with collection endpoints.
type PaginationMeta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	TotalCount  int `json:"total_count"`
	TotalPages  int `json:"total_pages"`
}

// doRequest performs an authenticated request and returns the raw response body.
// A non-2xx status is converted into an *APIError. body may be nil for requests
// without a payload (GET/DELETE).
func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.HostURL+path, reader)
	if err != nil {
		return nil, err
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("waiting for rate limiter: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= http.StatusBadRequest {
		return respBody, NewAPIError(res.StatusCode, respBody)
	}

	return respBody, nil
}

// getInto performs a GET and unmarshals the single-resource envelope into out.
func getInto[T any](ctx context.Context, c *Client, path string) (*T, error) {
	body, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[T](body)
}

// writeInto performs a write (POST/PATCH) and unmarshals the single-resource
// envelope into out.
func writeInto[T any](ctx context.Context, c *Client, method, path string, payload any) (*T, error) {
	body, err := c.doRequest(ctx, method, path, payload)
	if err != nil {
		return nil, err
	}
	return decodeData[T](body)
}

func decodeData[T any](body []byte) (*T, error) {
	var env dataEnvelope[T]
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &env.Data, nil
}

// listAll fetches every page of a paginated collection, following meta.total_pages.
func listAll[T any](ctx context.Context, c *Client, basePath string) ([]T, error) {
	separator := "?"
	if strings.Contains(basePath, "?") {
		separator = "&"
	}

	var all []T
	page := 1
	for {
		path := fmt.Sprintf("%s%spage=%d&per_page=100", basePath, separator, page)
		body, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var env listEnvelope[T]
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("decoding list response: %w", err)
		}
		all = append(all, env.Data...)

		// Stop when the API reports no further pages, or when it does not
		// paginate at all (TotalPages == 0 implies a non-paginated endpoint).
		if env.Meta.TotalPages <= page {
			break
		}
		page++
	}

	return all, nil
}

// deleteResource issues a DELETE; a 204 with an empty body is treated as success.
func (c *Client) deleteResource(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}
