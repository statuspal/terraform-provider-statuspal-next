// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"net/http"
	"net/url"
)

func servicesPath(subdomain string) string {
	return statusPagePath(subdomain) + "/services"
}

func servicePath(subdomain, slug string) string {
	return servicesPath(subdomain) + "/" + url.PathEscape(slug)
}

// ListServices returns the services on a status page.
func (c *Client) ListServices(ctx context.Context, subdomain string) ([]Service, error) {
	return listAll[Service](ctx, c, servicesPath(subdomain))
}

// GetService returns a single service by slug.
func (c *Client) GetService(ctx context.Context, subdomain, slug string) (*Service, error) {
	return getInto[Service](ctx, c, servicePath(subdomain, slug))
}

// CreateService creates a service. ContainerStatuses, when set, seeds the
// per-container initial statuses (containers not listed default to "ok").
func (c *Client) CreateService(ctx context.Context, subdomain string, svc *Service) (*Service, error) {
	return writeInto[Service](ctx, c, http.MethodPost, servicesPath(subdomain), svc)
}

// UpdateService updates a service identified by its current slug.
func (c *Client) UpdateService(ctx context.Context, subdomain, slug string, svc *Service) (*Service, error) {
	return writeInto[Service](ctx, c, http.MethodPatch, servicePath(subdomain, slug), svc)
}

// DeleteService deletes a service identified by slug.
func (c *Client) DeleteService(ctx context.Context, subdomain, slug string) error {
	return c.deleteResource(ctx, servicePath(subdomain, slug))
}
