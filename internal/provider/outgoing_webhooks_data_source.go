// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type outgoingWebhookDataModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	Events               types.Set    `tfsdk:"events"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	AllStatusPages       types.Bool   `tfsdk:"all_status_pages"`
	StatusPageSubdomains types.Set    `tfsdk:"status_page_subdomains"`
	LastTriggeredAt      types.String `tfsdk:"last_triggered_at"`
	LastAttemptedAt      types.String `tfsdk:"last_attempted_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func webhookToDataModel(ctx context.Context, wh *statuspalnext.OutgoingWebhook) (outgoingWebhookDataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	events, eventsDiags := types.SetValueFrom(ctx, types.StringType, wh.Events)
	diags.Append(eventsDiags...)

	model := outgoingWebhookDataModel{
		ID:              types.StringValue(wh.ID),
		Name:            types.StringValue(wh.Name),
		URL:             types.StringValue(wh.URL),
		Events:          events,
		Enabled:         types.BoolValue(boolValue(wh.Enabled, true)),
		AllStatusPages:  types.BoolValue(boolValue(wh.AllStatusPages, true)),
		LastTriggeredAt: optionalTimestamp(wh.LastTriggeredAt),
		LastAttemptedAt: optionalTimestamp(wh.LastAttemptedAt),
		CreatedAt:       types.StringValue(wh.CreatedAt),
		UpdatedAt:       types.StringValue(wh.UpdatedAt),
	}

	if boolValue(wh.AllStatusPages, true) {
		model.StatusPageSubdomains = types.SetNull(types.StringType)
	} else {
		subdomains, subDiags := types.SetValueFrom(ctx, types.StringType, wh.StatusPageSubdomains)
		diags.Append(subDiags...)
		model.StatusPageSubdomains = subdomains
	}

	return model, diags
}

var (
	_ datasource.DataSource              = &outgoingWebhooksDataSource{}
	_ datasource.DataSourceWithConfigure = &outgoingWebhooksDataSource{}
)

// NewOutgoingWebhooksDataSource is a helper function to simplify the provider implementation.
func NewOutgoingWebhooksDataSource() datasource.DataSource {
	return &outgoingWebhooksDataSource{}
}

type outgoingWebhooksDataSource struct {
	client *statuspalnext.Client
}

type outgoingWebhooksDataSourceModel struct {
	OutgoingWebhooks []outgoingWebhookDataModel `tfsdk:"outgoing_webhooks"`
}

func (d *outgoingWebhooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_outgoing_webhooks"
}

func (d *outgoingWebhooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all outgoing webhooks in the organization. The signing secret is never returned.",
		Attributes: map[string]schema.Attribute{
			"outgoing_webhooks": schema.ListNestedAttribute{
				MarkdownDescription: "All outgoing webhooks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.StringAttribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"url":                    schema.StringAttribute{Computed: true},
						"events":                 schema.SetAttribute{Computed: true, ElementType: types.StringType},
						"enabled":                schema.BoolAttribute{Computed: true},
						"all_status_pages":       schema.BoolAttribute{Computed: true},
						"status_page_subdomains": schema.SetAttribute{Computed: true, ElementType: types.StringType},
						"last_triggered_at":      schema.StringAttribute{Computed: true},
						"last_attempted_at":      schema.StringAttribute{Computed: true},
						"created_at":             schema.StringAttribute{Computed: true},
						"updated_at":             schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *outgoingWebhooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *outgoingWebhooksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	webhooks, err := d.client.ListOutgoingWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing outgoing webhooks", err.Error())
		return
	}

	state := outgoingWebhooksDataSourceModel{OutgoingWebhooks: make([]outgoingWebhookDataModel, 0, len(webhooks))}
	for i := range webhooks {
		model, diags := webhookToDataModel(ctx, &webhooks[i])
		resp.Diagnostics.Append(diags...)
		state.OutgoingWebhooks = append(state.OutgoingWebhooks, model)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
