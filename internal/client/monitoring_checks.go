// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

import (
	"context"
	"net/http"
	"net/url"
)

func monitoringCheckPath(id string) string {
	return "/monitoring_checks/" + url.PathEscape(id)
}

// ListMonitoringChecks returns every monitoring check in the organization.
func (c *Client) ListMonitoringChecks(ctx context.Context) ([]MonitoringCheck, error) {
	return listAll[MonitoringCheck](ctx, c, "/monitoring_checks")
}

// GetMonitoringCheck returns a single monitoring check by TypeID.
func (c *Client) GetMonitoringCheck(ctx context.Context, id string) (*MonitoringCheck, error) {
	return getInto[MonitoringCheck](ctx, c, monitoringCheckPath(id))
}

// CreateMonitoringCheck creates a monitoring check.
func (c *Client) CreateMonitoringCheck(ctx context.Context, check *MonitoringCheck) (*MonitoringCheck, error) {
	return writeInto[MonitoringCheck](ctx, c, http.MethodPost, "/monitoring_checks", check)
}

// UpdateMonitoringCheck updates a monitoring check by TypeID.
func (c *Client) UpdateMonitoringCheck(ctx context.Context, id string, check *MonitoringCheck) (*MonitoringCheck, error) {
	return writeInto[MonitoringCheck](ctx, c, http.MethodPatch, monitoringCheckPath(id), check)
}

// DeleteMonitoringCheck deletes a monitoring check by TypeID.
func (c *Client) DeleteMonitoringCheck(ctx context.Context, id string) error {
	return c.deleteResource(ctx, monitoringCheckPath(id))
}
