// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	monitoringCheckHTTPMethods = []string{"get", "head", "post", "put", "delete", "patch", "options"}
	monitoringCheckGeoAreas    = []string{"US", "EU"}
	monitoringCheckTimeouts    = []int64{1, 5, 10, 15, 30, 60}
)

var (
	_ resource.Resource                   = &monitoringCheckResource{}
	_ resource.ResourceWithConfigure      = &monitoringCheckResource{}
	_ resource.ResourceWithImportState    = &monitoringCheckResource{}
	_ resource.ResourceWithValidateConfig = &monitoringCheckResource{}
)

// NewMonitoringCheckResource is a helper function to simplify the provider implementation.
func NewMonitoringCheckResource() resource.Resource {
	return &monitoringCheckResource{}
}

type monitoringCheckResource struct {
	client *statuspalnext.Client
}

type monitoringCheckAutomationModel struct {
	StatusPageSubdomain types.String `tfsdk:"status_page_subdomain"`
	ContainerSlug       types.String `tfsdk:"container_slug"`
	ServiceSlug         types.String `tfsdk:"service_slug"`
}

type monitoringCheckResourceModel struct {
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

func (r *monitoringCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_check"
}

func (r *monitoringCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an uptime monitoring check (HTTP or TCP). Optionally drives status-page " +
			"incident automation: set the `automation` block to link the check to a status-page service so " +
			"that going down automatically opens an incident and recovering resolves it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "TypeID-prefixed identifier (e.g. `mck_…`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the check.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Full target URL. The scheme selects the check type: `tcp://host:port` is a " +
					"TCP check; `http://` or `https://` is an HTTP check.",
				Required: true,
			},
			"check_type": schema.StringAttribute{
				MarkdownDescription: "Check type derived from `url` (`http` or `tcp`).",
				Computed:            true,
			},
			"http_method": schema.StringAttribute{
				MarkdownDescription: "HTTP method for HTTP checks. Ignored for TCP checks (do not set it on a `tcp://` " +
					"check). One of: " + strings.Join(monitoringCheckHTTPMethods, ", ") + ".",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(monitoringCheckHTTPMethods...),
				},
			},
			"recv_timeout_secs": schema.Int64Attribute{
				MarkdownDescription: "Response timeout in seconds. One of: 1, 5, 10, 15, 30, 60.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(monitoringCheckTimeouts...),
				},
			},
			"geo_areas": schema.SetAttribute{
				MarkdownDescription: "Regions to monitor from (`US`, `EU`). Empty means all regions.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(monitoringCheckGeoAreas...)),
				},
			},
			"recipient_emails": schema.SetAttribute{
				MarkdownDescription: "Emails of organization users to notify on status changes. At least one required.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"display_response_time_chart": schema.BoolAttribute{
				MarkdownDescription: "Whether to show a response-time chart on the status page.",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Last status reported by the monitoring service (e.g. `up`, `down`); null until verified.",
				Computed:            true,
			},
			"automation": schema.SingleNestedAttribute{
				MarkdownDescription: "Links the check to a status-page service for incident automation. Omit to disable.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"status_page_subdomain": schema.StringAttribute{
						MarkdownDescription: "Subdomain of the status page whose service this check drives.",
						Required:            true,
					},
					"container_slug": schema.StringAttribute{
						MarkdownDescription: "Slug of the container (region) holding the target service.",
						Required:            true,
					},
					"service_slug": schema.StringAttribute{
						MarkdownDescription: "Slug of the service whose status this check updates.",
						Required:            true,
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the check was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the check was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *monitoringCheckResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *monitoringCheckResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config monitoringCheckResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.URL.IsUnknown() || config.HTTPMethod.IsUnknown() {
		return
	}

	isTCP := strings.HasPrefix(strings.ToLower(config.URL.ValueString()), "tcp://")
	if isTCP && !config.HTTPMethod.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("http_method"),
			"Invalid Attribute Combination",
			"http_method cannot be set for a TCP check. It only applies to http:// and https:// URLs; "+
				"remove http_method or change url to an http(s) scheme.",
		)
	}
}

func (r *monitoringCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitoringCheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := monitoringCheckRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMonitoringCheck(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitoring check", err.Error())
		return
	}

	model, diags := mapMonitoringCheckToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *monitoringCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitoringCheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	check, err := r.client.GetMonitoringCheck(ctx, state.ID.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitoring check", err.Error())
		return
	}

	model, diags := mapMonitoringCheckToModel(ctx, check)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *monitoringCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitoringCheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := monitoringCheckRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateMonitoringCheck(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitoring check", err.Error())
		return
	}

	model, diags := mapMonitoringCheckToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *monitoringCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitoringCheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMonitoringCheck(ctx, state.ID.ValueString())
	if err != nil && !statuspalnext.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting monitoring check", err.Error())
	}
}

func (r *monitoringCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func monitoringCheckRequestBody(ctx context.Context, plan *monitoringCheckResourceModel) (*statuspalnext.MonitoringCheck, diag.Diagnostics) {
	var diags diag.Diagnostics

	check := &statuspalnext.MonitoringCheck{
		Name: plan.Name.ValueString(),
		URL:  plan.URL.ValueString(),
	}

	isTCP := strings.HasPrefix(strings.ToLower(plan.URL.ValueString()), "tcp://")
	// http_method is only meaningful for HTTP checks; never send it for TCP.
	if !isTCP && !plan.HTTPMethod.IsNull() && !plan.HTTPMethod.IsUnknown() {
		check.HTTPMethod = statuspalnext.StringPtr(plan.HTTPMethod.ValueString())
	}
	if !plan.RecvTimeoutSecs.IsNull() && !plan.RecvTimeoutSecs.IsUnknown() {
		check.RecvTimeoutSecs = statuspalnext.Int64Ptr(plan.RecvTimeoutSecs.ValueInt64())
	}
	if !plan.DisplayResponseTimeChart.IsNull() && !plan.DisplayResponseTimeChart.IsUnknown() {
		check.DisplayResponseTimeChart = statuspalnext.BoolPtr(plan.DisplayResponseTimeChart.ValueBool())
	}
	if !plan.GeoAreas.IsNull() && !plan.GeoAreas.IsUnknown() {
		var geoAreas []string
		diags.Append(plan.GeoAreas.ElementsAs(ctx, &geoAreas, false)...)
		check.GeoAreas = geoAreas
	}

	var recipients []string
	diags.Append(plan.RecipientEmails.ElementsAs(ctx, &recipients, false)...)
	check.RecipientEmails = recipients

	// Automation is always sent so an absent block clears it (explicit null disables).
	if plan.Automation != nil {
		check.Automation = &statuspalnext.MonitoringCheckAutomation{
			StatusPageSubdomain: plan.Automation.StatusPageSubdomain.ValueString(),
			ContainerSlug:       plan.Automation.ContainerSlug.ValueString(),
			ServiceSlug:         plan.Automation.ServiceSlug.ValueString(),
		}
	}

	return check, diags
}

func mapMonitoringCheckToModel(ctx context.Context, check *statuspalnext.MonitoringCheck) (monitoringCheckResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	geoAreas, geoDiags := types.SetValueFrom(ctx, types.StringType, check.GeoAreas)
	diags.Append(geoDiags...)
	recipients, recDiags := types.SetValueFrom(ctx, types.StringType, check.RecipientEmails)
	diags.Append(recDiags...)

	model := monitoringCheckResourceModel{
		ID:                       types.StringValue(check.ID),
		Name:                     types.StringValue(check.Name),
		URL:                      types.StringValue(check.URL),
		CheckType:                types.StringValue(check.CheckType),
		RecvTimeoutSecs:          types.Int64Value(int64Value(check.RecvTimeoutSecs)),
		GeoAreas:                 geoAreas,
		RecipientEmails:          recipients,
		DisplayResponseTimeChart: types.BoolValue(boolValue(check.DisplayResponseTimeChart, false)),
		Status:                   optionalString(check.Status),
		CreatedAt:                types.StringValue(check.CreatedAt),
		UpdatedAt:                types.StringValue(check.UpdatedAt),
	}

	if check.HTTPMethod != nil && *check.HTTPMethod != "" {
		model.HTTPMethod = types.StringValue(*check.HTTPMethod)
	} else {
		model.HTTPMethod = types.StringNull()
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
