// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// statusPageDataModel is the shared, read-only projection of a status page used
// by both the singular and plural data sources.
type statusPageDataModel struct {
	ID                    types.String               `tfsdk:"id"`
	Name                  types.String               `tfsdk:"name"`
	Subdomain             types.String               `tfsdk:"subdomain"`
	Timezone              types.String               `tfsdk:"timezone"`
	WebsiteURL            types.String               `tfsdk:"website_url"`
	RequireAuthentication types.Bool                 `tfsdk:"require_authentication"`
	CurrentStatus         types.String               `tfsdk:"current_status"`
	NotificationSettings  *notificationSettingsModel `tfsdk:"notification_settings"`
	CreatedAt             types.String               `tfsdk:"created_at"`
	UpdatedAt             types.String               `tfsdk:"updated_at"`
}

func statusPageToDataModel(sp *statuspalnext.StatusPage) statusPageDataModel {
	ns := sp.NotificationSettings
	if ns == nil {
		ns = &statuspalnext.NotificationSettings{}
	}
	return statusPageDataModel{
		ID:                    types.StringValue(sp.ID),
		Name:                  types.StringValue(sp.Name),
		Subdomain:             types.StringValue(sp.Subdomain),
		Timezone:              types.StringValue(sp.Timezone),
		WebsiteURL:            types.StringValue(sp.WebsiteURL),
		RequireAuthentication: types.BoolValue(boolValue(sp.RequireAuthentication, false)),
		CurrentStatus:         types.StringValue(sp.CurrentStatus),
		NotificationSettings: &notificationSettingsModel{
			EmailEnabled:   types.BoolValue(boolValue(ns.EmailEnabled, true)),
			SlackEnabled:   types.BoolValue(boolValue(ns.SlackEnabled, false)),
			RSSFeedEnabled: types.BoolValue(boolValue(ns.RSSFeedEnabled, true)),
		},
		CreatedAt: types.StringValue(sp.CreatedAt),
		UpdatedAt: types.StringValue(sp.UpdatedAt),
	}
}

// statusPageDataSourceAttributes returns the read-only attribute set shared by
// both data sources. subdomainRequired controls whether `subdomain` is an input
// (singular data source) or an output (plural data source).
func statusPageDataSourceAttributes(subdomainRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                     schema.StringAttribute{Computed: true, MarkdownDescription: "Status page identifier."},
		"name":                   schema.StringAttribute{Computed: true, MarkdownDescription: "Display name."},
		"subdomain":              schema.StringAttribute{Required: subdomainRequired, Computed: !subdomainRequired, MarkdownDescription: "Status page subdomain."},
		"timezone":               schema.StringAttribute{Computed: true, MarkdownDescription: "Primary timezone."},
		"website_url":            schema.StringAttribute{Computed: true, MarkdownDescription: "Reported website URL."},
		"require_authentication": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether viewing requires sign-in."},
		"current_status":         schema.StringAttribute{Computed: true, MarkdownDescription: "Derived overall status."},
		"notification_settings": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Subscriber notification channels.",
			Attributes: map[string]schema.Attribute{
				"email_enabled":    schema.BoolAttribute{Computed: true},
				"slack_enabled":    schema.BoolAttribute{Computed: true},
				"rss_feed_enabled": schema.BoolAttribute{Computed: true},
			},
		},
		"created_at": schema.StringAttribute{Computed: true, MarkdownDescription: "Creation timestamp."},
		"updated_at": schema.StringAttribute{Computed: true, MarkdownDescription: "Last update timestamp."},
	}
}

var (
	_ datasource.DataSource              = &statusPagesDataSource{}
	_ datasource.DataSourceWithConfigure = &statusPagesDataSource{}
)

// NewStatusPagesDataSource is a helper function to simplify the provider implementation.
func NewStatusPagesDataSource() datasource.DataSource {
	return &statusPagesDataSource{}
}

type statusPagesDataSource struct {
	client *statuspalnext.Client
}

type statusPagesDataSourceModel struct {
	StatusPages []statusPageDataModel `tfsdk:"status_pages"`
}

func (d *statusPagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_pages"
}

func (d *statusPagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all status pages in the organization.",
		Attributes: map[string]schema.Attribute{
			"status_pages": schema.ListNestedAttribute{
				MarkdownDescription: "All status pages.",
				Computed:            true,
				NestedObject:        schema.NestedAttributeObject{Attributes: statusPageDataSourceAttributes(false)},
			},
		},
	}
}

func (d *statusPagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *statusPagesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	pages, err := d.client.ListStatusPages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing status pages", err.Error())
		return
	}

	state := statusPagesDataSourceModel{StatusPages: make([]statusPageDataModel, 0, len(pages))}
	for i := range pages {
		state.StatusPages = append(state.StatusPages, statusPageToDataModel(&pages[i]))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
