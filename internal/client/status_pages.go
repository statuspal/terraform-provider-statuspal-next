// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ListStatusPages returns every status page in the organization.
func (c *Client) ListStatusPages(ctx context.Context) ([]StatusPage, error) {
	return listAll[StatusPage](ctx, c, "/status_pages")
}

// GetStatusPage returns a single status page by subdomain.
func (c *Client) GetStatusPage(ctx context.Context, subdomain string) (*StatusPage, error) {
	return getInto[StatusPage](ctx, c, "/status_pages/"+url.PathEscape(subdomain))
}

// CreateStatusPage creates a status page.
func (c *Client) CreateStatusPage(ctx context.Context, sp *StatusPage) (*StatusPage, error) {
	return writeInto[StatusPage](ctx, c, http.MethodPost, "/status_pages", sp)
}

// UpdateStatusPage updates the status page identified by subdomain.
func (c *Client) UpdateStatusPage(ctx context.Context, subdomain string, sp *StatusPage) (*StatusPage, error) {
	return writeInto[StatusPage](ctx, c, http.MethodPatch, "/status_pages/"+url.PathEscape(subdomain), sp)
}

// DeleteStatusPage deletes the status page identified by subdomain.
func (c *Client) DeleteStatusPage(ctx context.Context, subdomain string) error {
	return c.deleteResource(ctx, "/status_pages/"+url.PathEscape(subdomain))
}

func statusPagePath(subdomain string) string {
	return fmt.Sprintf("/status_pages/%s", url.PathEscape(subdomain))
}
