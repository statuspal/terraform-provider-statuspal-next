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

type automationDataModel struct {
	ID                 types.String `tfsdk:"id"`
	ServiceSlug        types.String `tfsdk:"service_slug"`
	ContainerSlug      types.String `tfsdk:"container_slug"`
	ManageIncidents    types.Bool   `tfsdk:"manage_incidents"`
	HasSecret          types.Bool   `tfsdk:"has_secret"`
	ExpectedResultPath types.String `tfsdk:"expected_result_path"`
	ExpectedResult     types.String `tfsdk:"expected_result"`
	SecretPath         types.String `tfsdk:"secret_path"`
	TriggerURL         types.String `tfsdk:"trigger_url"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func automationToDataModel(a *statuspalnext.Automation) automationDataModel {
	model := automationDataModel{
		ID:              types.StringValue(a.ID),
		ServiceSlug:     types.StringValue(a.ServiceSlug),
		ContainerSlug:   types.StringValue(a.ContainerSlug),
		ManageIncidents: types.BoolValue(boolValue(a.ManageIncidents, false)),
		HasSecret:       types.BoolValue(a.HasSecret),
		TriggerURL:      types.StringValue(a.TriggerURL),
		CreatedAt:       types.StringValue(a.CreatedAt),
		UpdatedAt:       types.StringValue(a.UpdatedAt),
	}
	if a.AutomationFormat != nil {
		model.ExpectedResultPath = types.StringValue(a.AutomationFormat.ExpectedResultPath)
		model.ExpectedResult = types.StringValue(a.AutomationFormat.ExpectedResult)
		model.SecretPath = optionalString(a.AutomationFormat.SecretPath)
	}
	return model
}

var (
	_ datasource.DataSource              = &automationsDataSource{}
	_ datasource.DataSourceWithConfigure = &automationsDataSource{}
)

// NewAutomationsDataSource is a helper function to simplify the provider implementation.
func NewAutomationsDataSource() datasource.DataSource {
	return &automationsDataSource{}
}

type automationsDataSource struct {
	client *statuspalnext.Client
}

type automationsDataSourceModel struct {
	StatusPageSubdomain types.String          `tfsdk:"status_page_subdomain"`
	Automations         []automationDataModel `tfsdk:"automations"`
}

func (d *automationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automations"
}

func (d *automationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the incident automations on a status page.",
		Attributes: map[string]schema.Attribute{
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page whose automations to list.",
				Required:            true,
			},
			"automations": schema.ListNestedAttribute{
				MarkdownDescription: "Automations on the status page.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                   schema.StringAttribute{Computed: true},
						"service_slug":         schema.StringAttribute{Computed: true},
						"container_slug":       schema.StringAttribute{Computed: true},
						"manage_incidents":     schema.BoolAttribute{Computed: true},
						"has_secret":           schema.BoolAttribute{Computed: true},
						"expected_result_path": schema.StringAttribute{Computed: true},
						"expected_result":      schema.StringAttribute{Computed: true},
						"secret_path":          schema.StringAttribute{Computed: true},
						"trigger_url":          schema.StringAttribute{Computed: true, Sensitive: true},
						"created_at":           schema.StringAttribute{Computed: true},
						"updated_at":           schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *automationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *automationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config automationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	automations, err := d.client.ListAutomations(ctx, config.StatusPageSubdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing automations", err.Error())
		return
	}

	config.Automations = make([]automationDataModel, 0, len(automations))
	for i := range automations {
		config.Automations = append(config.Automations, automationToDataModel(&automations[i]))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
