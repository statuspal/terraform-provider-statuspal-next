// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &automationResource{}
	_ resource.ResourceWithConfigure   = &automationResource{}
	_ resource.ResourceWithImportState = &automationResource{}
)

// NewAutomationResource is a helper function to simplify the provider implementation.
func NewAutomationResource() resource.Resource {
	return &automationResource{}
}

type automationResource struct {
	client *statuspalnext.Client
}

type automationFormatModel struct {
	ExpectedResultPath types.String `tfsdk:"expected_result_path"`
	ExpectedResult     types.String `tfsdk:"expected_result"`
	SecretPath         types.String `tfsdk:"secret_path"`
}

type automationResourceModel struct {
	ID                  types.String           `tfsdk:"id"`
	StatusPageSubdomain types.String           `tfsdk:"status_page_subdomain"`
	ServiceSlug         types.String           `tfsdk:"service_slug"`
	ContainerSlug       types.String           `tfsdk:"container_slug"`
	ManageIncidents     types.Bool             `tfsdk:"manage_incidents"`
	Secret              types.String           `tfsdk:"secret"`
	AutomationFormat    *automationFormatModel `tfsdk:"automation_format"`
	TriggerURL          types.String           `tfsdk:"trigger_url"`
	CreatedAt           types.String           `tfsdk:"created_at"`
	UpdatedAt           types.String           `tfsdk:"updated_at"`
}

func (r *automationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation"
}

func (r *automationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a webhook-driven incident automation on a status page. An external monitor " +
			"POSTs its JSON payload to the generated `trigger_url`; StatusPal evaluates it against the " +
			"`automation_format` JSONPath rule and opens/closes an incident (or flips the service status).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "TypeID-prefixed identifier (e.g. `auto_…`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page this automation belongs to. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_slug": schema.StringAttribute{
				MarkdownDescription: "Slug of the service this automation updates.",
				Required:            true,
			},
			"container_slug": schema.StringAttribute{
				MarkdownDescription: "Slug of the container (region) the service belongs to.",
				Required:            true,
			},
			"manage_incidents": schema.BoolAttribute{
				MarkdownDescription: "When true, triggers open/close incidents; when false, they only flip the service status.",
				Optional:            true,
				Computed:            true,
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Optional secret used to validate incoming trigger requests (matched against " +
					"`automation_format.secret_path`). Write-only — never returned by the API; kept in state.",
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"automation_format": schema.SingleNestedAttribute{
				MarkdownDescription: "Custom JSONPath rule used to interpret trigger payloads.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"expected_result_path": schema.StringAttribute{
						MarkdownDescription: "JSONPath evaluated against the trigger payload (e.g. `$.status`).",
						Required:            true,
					},
					"expected_result": schema.StringAttribute{
						MarkdownDescription: "Value the path must equal (case-insensitive) for the service to be healthy.",
						Required:            true,
					},
					"secret_path": schema.StringAttribute{
						MarkdownDescription: "Optional JSONPath (`$.token`) or header (`h:X-Secret`) the request secret " +
							"is read from. Required when `secret` is set.",
						Optional: true,
					},
				},
			},
			"trigger_url": schema.StringAttribute{
				MarkdownDescription: "URL external monitors POST trigger payloads to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the automation was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the automation was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAutomation(ctx, plan.StatusPageSubdomain.ValueString(), automationRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating automation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(created, &plan))...)
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	automation, err := r.client.GetAutomation(ctx, state.StatusPageSubdomain.ValueString(), state.ID.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading automation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(automation, &state))...)
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state automationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAutomation(ctx, state.StatusPageSubdomain.ValueString(), state.ID.ValueString(), automationRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating automation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(updated, &plan))...)
}

func (r *automationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAutomation(ctx, state.StatusPageSubdomain.ValueString(), state.ID.ValueString())
	if err != nil && !statuspalnext.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting automation", err.Error())
	}
}

// ImportState expects "<status_page_subdomain>/<automation_id>".
func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			`Expected import ID in the format "<status_page_subdomain>/<automation_id>".`,
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_subdomain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func automationRequestBody(plan *automationResourceModel) *statuspalnext.Automation {
	a := &statuspalnext.Automation{
		ServiceSlug:     plan.ServiceSlug.ValueString(),
		ContainerSlug:   plan.ContainerSlug.ValueString(),
		ManageIncidents: statuspalnext.BoolPtr(plan.ManageIncidents.ValueBool()),
		AutomationFormat: &statuspalnext.AutomationFormat{
			ExpectedResultPath: plan.AutomationFormat.ExpectedResultPath.ValueString(),
			ExpectedResult:     plan.AutomationFormat.ExpectedResult.ValueString(),
		},
	}
	if !plan.AutomationFormat.SecretPath.IsNull() && !plan.AutomationFormat.SecretPath.IsUnknown() {
		a.AutomationFormat.SecretPath = statuspalnext.StringPtr(plan.AutomationFormat.SecretPath.ValueString())
	}
	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		a.Secret = statuspalnext.StringPtr(plan.Secret.ValueString())
	}
	return a
}

// automationToModel maps an API response into the tfsdk model. prior carries the
// write-only secret, which the API never returns.
func automationToModel(a *statuspalnext.Automation, prior *automationResourceModel) automationResourceModel {
	model := automationResourceModel{
		ID:                  types.StringValue(a.ID),
		StatusPageSubdomain: prior.StatusPageSubdomain,
		ServiceSlug:         types.StringValue(a.ServiceSlug),
		ContainerSlug:       types.StringValue(a.ContainerSlug),
		ManageIncidents:     types.BoolValue(boolValue(a.ManageIncidents, false)),
		Secret:              prior.Secret,
		TriggerURL:          types.StringValue(a.TriggerURL),
		CreatedAt:           types.StringValue(a.CreatedAt),
		UpdatedAt:           types.StringValue(a.UpdatedAt),
	}

	if a.AutomationFormat != nil {
		model.AutomationFormat = &automationFormatModel{
			ExpectedResultPath: types.StringValue(a.AutomationFormat.ExpectedResultPath),
			ExpectedResult:     types.StringValue(a.AutomationFormat.ExpectedResult),
			SecretPath:         optionalString(a.AutomationFormat.SecretPath),
		}
	}

	return model
}
