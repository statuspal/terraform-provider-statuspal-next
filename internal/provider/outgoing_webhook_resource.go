// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// webhookEvents is the set of event names the API accepts.
var webhookEvents = []string{
	"service.status_changed",
	"notice.created",
	"notice.update_posted",
	"notice.updated",
}

var (
	_ resource.Resource                   = &outgoingWebhookResource{}
	_ resource.ResourceWithConfigure      = &outgoingWebhookResource{}
	_ resource.ResourceWithImportState    = &outgoingWebhookResource{}
	_ resource.ResourceWithValidateConfig = &outgoingWebhookResource{}
)

// NewOutgoingWebhookResource is a helper function to simplify the provider implementation.
func NewOutgoingWebhookResource() resource.Resource {
	return &outgoingWebhookResource{}
}

type outgoingWebhookResource struct {
	client *statuspalnext.Client
}

type outgoingWebhookResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	Events               types.Set    `tfsdk:"events"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	AllStatusPages       types.Bool   `tfsdk:"all_status_pages"`
	StatusPageSubdomains types.Set    `tfsdk:"status_page_subdomains"`
	Secret               types.String `tfsdk:"secret"`
	LastTriggeredAt      types.String `tfsdk:"last_triggered_at"`
	LastAttemptedAt      types.String `tfsdk:"last_attempted_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *outgoingWebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_outgoing_webhook"
}

func (r *outgoingWebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an organization-level outgoing webhook subscription.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "TypeID-prefixed identifier (e.g. `wbk_…`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the webhook.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "HTTPS URL that receives `POST` deliveries.",
				Required:            true,
			},
			"events": schema.SetAttribute{
				MarkdownDescription: "Events that trigger a delivery. One or more of: " +
					"`service.status_changed`, `notice.created`, `notice.update_posted`, `notice.updated`.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(webhookEvents...)),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the webhook is active.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"all_status_pages": schema.BoolAttribute{
				MarkdownDescription: "When `true`, fires for events on every status page in the organization " +
					"(including pages created later). When `false`, only the pages in " +
					"`status_page_subdomains` fire.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"status_page_subdomains": schema.SetAttribute{
				MarkdownDescription: "Subdomains of the status pages this webhook is scoped to. " +
					"Required when `all_status_pages` is `false`; ignored otherwise.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Plaintext signing secret (`whs_…`), returned only when the webhook is " +
					"created. Stored in state so signature verification can use it; it cannot be retrieved " +
					"from the API afterwards.",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_triggered_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the most recent successful delivery.",
				Computed:            true,
			},
			"last_attempted_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the most recent delivery attempt.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the webhook was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the webhook was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *outgoingWebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *outgoingWebhookResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config outgoingWebhookResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.AllStatusPages.IsUnknown() || config.StatusPageSubdomains.IsUnknown() {
		return
	}

	allStatusPages := config.AllStatusPages.IsNull() || config.AllStatusPages.ValueBool()
	hasSubdomains := !config.StatusPageSubdomains.IsNull() && len(config.StatusPageSubdomains.Elements()) > 0

	switch {
	case allStatusPages && hasSubdomains:
		resp.Diagnostics.AddAttributeError(
			path.Root("status_page_subdomains"),
			"Invalid Attribute Combination",
			"status_page_subdomains cannot be set when all_status_pages is true. "+
				"Set all_status_pages to false to scope the webhook to specific pages, or remove status_page_subdomains.",
		)
	case !allStatusPages && !hasSubdomains:
		resp.Diagnostics.AddAttributeError(
			path.Root("status_page_subdomains"),
			"Missing Attribute Configuration",
			"status_page_subdomains is required when all_status_pages is false. "+
				"List the status page subdomains this webhook should be scoped to.",
		)
	}
}

func (r *outgoingWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan outgoingWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := webhookRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateOutgoingWebhook(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating outgoing webhook", err.Error())
		return
	}

	model, diags := mapWebhookToModel(ctx, created, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *outgoingWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state outgoingWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, err := r.client.GetOutgoingWebhook(ctx, state.ID.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading outgoing webhook", err.Error())
		return
	}

	// The secret is not returned on read; preserve the value already in state.
	model, diags := mapWebhookToModel(ctx, wh, state.Secret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *outgoingWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state outgoingWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := webhookRequestBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateOutgoingWebhook(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating outgoing webhook", err.Error())
		return
	}

	// The secret is not returned on update; preserve the value already in state.
	model, diags := mapWebhookToModel(ctx, updated, state.Secret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *outgoingWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state outgoingWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOutgoingWebhook(ctx, state.ID.ValueString())
	if err != nil && !statuspalnext.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting outgoing webhook", err.Error())
	}
}

func (r *outgoingWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func webhookRequestBody(ctx context.Context, plan *outgoingWebhookResourceModel) (*statuspalnext.OutgoingWebhook, diag.Diagnostics) {
	var diags diag.Diagnostics

	var events []string
	diags.Append(plan.Events.ElementsAs(ctx, &events, false)...)

	wh := &statuspalnext.OutgoingWebhook{
		Name:           plan.Name.ValueString(),
		URL:            plan.URL.ValueString(),
		Events:         events,
		Enabled:        statuspalnext.BoolPtr(plan.Enabled.ValueBool()),
		AllStatusPages: statuspalnext.BoolPtr(plan.AllStatusPages.ValueBool()),
	}

	// Subdomains only matter when the webhook is scoped to specific pages. When
	// all_status_pages is true, the API clears the association regardless.
	if !plan.AllStatusPages.ValueBool() && !plan.StatusPageSubdomains.IsNull() && !plan.StatusPageSubdomains.IsUnknown() {
		var subdomains []string
		diags.Append(plan.StatusPageSubdomains.ElementsAs(ctx, &subdomains, false)...)
		wh.StatusPageSubdomains = subdomains
	}

	return wh, diags
}

func mapWebhookToModel(ctx context.Context, wh *statuspalnext.OutgoingWebhook, priorSecret types.String) (outgoingWebhookResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	events, eventsDiags := types.SetValueFrom(ctx, types.StringType, wh.Events)
	diags.Append(eventsDiags...)

	model := outgoingWebhookResourceModel{
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

	// status_page_subdomains is only meaningful (and only returned) when the
	// webhook is scoped to specific pages.
	if boolValue(wh.AllStatusPages, true) {
		model.StatusPageSubdomains = types.SetNull(types.StringType)
	} else {
		subdomains, subDiags := types.SetValueFrom(ctx, types.StringType, wh.StatusPageSubdomains)
		diags.Append(subDiags...)
		model.StatusPageSubdomains = subdomains
	}

	// The secret is only present in create/regenerate responses; otherwise keep
	// whatever was already stored.
	if wh.Secret != "" {
		model.Secret = types.StringValue(wh.Secret)
	} else {
		model.Secret = priorSecret
	}

	return model, diags
}

func optionalTimestamp(p *string) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
