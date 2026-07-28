// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"net/http"
	"net/url"
)

func outgoingWebhookPath(id string) string {
	return "/outgoing_webhooks/" + url.PathEscape(id)
}

// ListOutgoingWebhooks returns every outgoing webhook in the organization.
func (c *Client) ListOutgoingWebhooks(ctx context.Context) ([]OutgoingWebhook, error) {
	return listAll[OutgoingWebhook](ctx, c, "/outgoing_webhooks")
}

// GetOutgoingWebhook returns a single webhook by TypeID. The signing secret is
// not included in this response (it is only returned on create/regenerate).
func (c *Client) GetOutgoingWebhook(ctx context.Context, id string) (*OutgoingWebhook, error) {
	return getInto[OutgoingWebhook](ctx, c, outgoingWebhookPath(id))
}

// CreateOutgoingWebhook creates a webhook. The returned webhook includes the
// plaintext Secret exactly once.
func (c *Client) CreateOutgoingWebhook(ctx context.Context, wh *OutgoingWebhook) (*OutgoingWebhook, error) {
	return writeInto[OutgoingWebhook](ctx, c, http.MethodPost, "/outgoing_webhooks", wh)
}

// UpdateOutgoingWebhook updates a webhook by TypeID.
func (c *Client) UpdateOutgoingWebhook(ctx context.Context, id string, wh *OutgoingWebhook) (*OutgoingWebhook, error) {
	return writeInto[OutgoingWebhook](ctx, c, http.MethodPatch, outgoingWebhookPath(id), wh)
}

// DeleteOutgoingWebhook deletes a webhook by TypeID.
func (c *Client) DeleteOutgoingWebhook(ctx context.Context, id string) error {
	return c.deleteResource(ctx, outgoingWebhookPath(id))
}
