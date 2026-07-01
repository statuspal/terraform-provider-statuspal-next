// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"net/http"
	"net/url"
)

func automationsPath(subdomain string) string {
	return "/status_pages/" + url.PathEscape(subdomain) + "/automations"
}

func automationPath(subdomain, id string) string {
	return automationsPath(subdomain) + "/" + url.PathEscape(id)
}

// ListAutomations returns every automation on a status page.
func (c *Client) ListAutomations(ctx context.Context, subdomain string) ([]Automation, error) {
	return listAll[Automation](ctx, c, automationsPath(subdomain))
}

// GetAutomation returns a single automation by TypeID.
func (c *Client) GetAutomation(ctx context.Context, subdomain, id string) (*Automation, error) {
	return getInto[Automation](ctx, c, automationPath(subdomain, id))
}

// CreateAutomation creates an automation on a status page.
func (c *Client) CreateAutomation(ctx context.Context, subdomain string, a *Automation) (*Automation, error) {
	return writeInto[Automation](ctx, c, http.MethodPost, automationsPath(subdomain), a)
}

// UpdateAutomation updates an automation by TypeID.
func (c *Client) UpdateAutomation(ctx context.Context, subdomain, id string, a *Automation) (*Automation, error) {
	return writeInto[Automation](ctx, c, http.MethodPatch, automationPath(subdomain, id), a)
}

// DeleteAutomation deletes an automation by TypeID.
func (c *Client) DeleteAutomation(ctx context.Context, subdomain, id string) error {
	return c.deleteResource(ctx, automationPath(subdomain, id))
}
