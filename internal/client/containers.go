// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"net/http"
	"net/url"
)

func containersPath(subdomain string) string {
	return statusPagePath(subdomain) + "/containers"
}

func containerPath(subdomain, slug string) string {
	return containersPath(subdomain) + "/" + url.PathEscape(slug)
}

// ListContainers returns the containers on a status page.
func (c *Client) ListContainers(ctx context.Context, subdomain string) ([]Container, error) {
	return listAll[Container](ctx, c, containersPath(subdomain))
}

// GetContainer returns a single container by slug.
func (c *Client) GetContainer(ctx context.Context, subdomain, slug string) (*Container, error) {
	return getInto[Container](ctx, c, containerPath(subdomain, slug))
}

// CreateContainer creates a container. Every existing service on the status page
// is automatically assigned to it (full N×M grid).
func (c *Client) CreateContainer(ctx context.Context, subdomain string, ct *Container) (*Container, error) {
	return writeInto[Container](ctx, c, http.MethodPost, containersPath(subdomain), ct)
}

// UpdateContainer updates a container identified by its current slug.
func (c *Client) UpdateContainer(ctx context.Context, subdomain, slug string, ct *Container) (*Container, error) {
	return writeInto[Container](ctx, c, http.MethodPatch, containerPath(subdomain, slug), ct)
}

// DeleteContainer deletes a container identified by slug.
func (c *Client) DeleteContainer(ctx context.Context, subdomain, slug string) error {
	return c.deleteResource(ctx, containerPath(subdomain, slug))
}
