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

type monitoringCheckDataModel struct {
	ID                       types.String                    `tfsdk:"id"`
	Name                     types.String                    `tfsdk:"name"`
	URL                      types.String                    `tfsdk:"url"`
	CheckType                types.String                    `tfsdk:"check_type"`
	HTTPMethod               types.String                    `tfsdk:"http_method"`
	RecvTimeoutSecs          types.Int64                     `tfsdk:"recv_timeout_secs"`
	GeoAreas                 types.Set                       `tfsdk:"geo_areas"`
	RecipientEmails          types.Set                       `tfsdk:"recipient_emails"`
	DisplayResponseTimeChart types.Bool                      `tfsdk:"display_response_time_chart"`
	Status                   types.String                    `tfsdk:"status"`
	Automation               *monitoringCheckAutomationModel `tfsdk:"automation"`
	CreatedAt                types.String                    `tfsdk:"created_at"`
	UpdatedAt                types.String                    `tfsdk:"updated_at"`
}

func monitoringCheckToDataModel(ctx context.Context, check *statuspalnext.MonitoringCheck) (monitoringCheckDataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	geoAreas, geoDiags := types.SetValueFrom(ctx, types.StringType, check.GeoAreas)
	diags.Append(geoDiags...)
	recipients, recDiags := types.SetValueFrom(ctx, types.StringType, check.RecipientEmails)
	diags.Append(recDiags...)

	model := monitoringCheckDataModel{
		ID:                       types.StringValue(check.ID),
		Name:                     types.StringValue(check.Name),
		URL:                      types.StringValue(check.URL),
		CheckType:                types.StringValue(check.CheckType),
		HTTPMethod:               optionalString(check.HTTPMethod),
		RecvTimeoutSecs:          types.Int64Value(int64Value(check.RecvTimeoutSecs)),
		GeoAreas:                 geoAreas,
		RecipientEmails:          recipients,
		DisplayResponseTimeChart: types.BoolValue(boolValue(check.DisplayResponseTimeChart, false)),
		Status:                   optionalString(check.Status),
		CreatedAt:                types.StringValue(check.CreatedAt),
		UpdatedAt:                types.StringValue(check.UpdatedAt),
	}

	if check.Automation != nil {
		model.Automation = &monitoringCheckAutomationModel{
			StatusPageSubdomain: types.StringValue(check.Automation.StatusPageSubdomain),
			ContainerSlug:       types.StringValue(check.Automation.ContainerSlug),
			ServiceSlug:         types.StringValue(check.Automation.ServiceSlug),
		}
	}

	return model, diags
}

var (
	_ datasource.DataSource              = &monitoringChecksDataSource{}
	_ datasource.DataSourceWithConfigure = &monitoringChecksDataSource{}
)

// NewMonitoringChecksDataSource is a helper function to simplify the provider implementation.
func NewMonitoringChecksDataSource() datasource.DataSource {
	return &monitoringChecksDataSource{}
}

type monitoringChecksDataSource struct {
	client *statuspalnext.Client
}

type monitoringChecksDataSourceModel struct {
	MonitoringChecks []monitoringCheckDataModel `tfsdk:"monitoring_checks"`
}

func (d *monitoringChecksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_checks"
}

func (d *monitoringChecksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	automationAttr := schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"status_page_subdomain": schema.StringAttribute{Computed: true},
			"container_slug":        schema.StringAttribute{Computed: true},
			"service_slug":          schema.StringAttribute{Computed: true},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all monitoring checks in the organization.",
		Attributes: map[string]schema.Attribute{
			"monitoring_checks": schema.ListNestedAttribute{
				MarkdownDescription: "All monitoring checks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                          schema.StringAttribute{Computed: true},
						"name":                        schema.StringAttribute{Computed: true},
						"url":                         schema.StringAttribute{Computed: true},
						"check_type":                  schema.StringAttribute{Computed: true},
						"http_method":                 schema.StringAttribute{Computed: true},
						"recv_timeout_secs":           schema.Int64Attribute{Computed: true},
						"geo_areas":                   schema.SetAttribute{Computed: true, ElementType: types.StringType},
						"recipient_emails":            schema.SetAttribute{Computed: true, ElementType: types.StringType},
						"display_response_time_chart": schema.BoolAttribute{Computed: true},
						"status":                      schema.StringAttribute{Computed: true},
						"automation":                  automationAttr,
						"created_at":                  schema.StringAttribute{Computed: true},
						"updated_at":                  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *monitoringChecksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *monitoringChecksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	checks, err := d.client.ListMonitoringChecks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing monitoring checks", err.Error())
		return
	}

	state := monitoringChecksDataSourceModel{MonitoringChecks: make([]monitoringCheckDataModel, 0, len(checks))}
	for i := range checks {
		model, diags := monitoringCheckToDataModel(ctx, &checks[i])
		resp.Diagnostics.Append(diags...)
		state.MonitoringChecks = append(state.MonitoringChecks, model)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
