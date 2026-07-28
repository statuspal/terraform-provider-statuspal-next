// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// notificationSettingsAttrTypes describes the notification_settings nested object.
var notificationSettingsAttrTypes = map[string]attr.Type{
	"email_enabled":    types.BoolType,
	"slack_enabled":    types.BoolType,
	"rss_feed_enabled": types.BoolType,
}

// defaultNotificationSettings matches the API defaults (email + rss on, slack off).
func defaultNotificationSettings() types.Object {
	return types.ObjectValueMust(notificationSettingsAttrTypes, map[string]attr.Value{
		"email_enabled":    types.BoolValue(true),
		"slack_enabled":    types.BoolValue(false),
		"rss_feed_enabled": types.BoolValue(true),
	})
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &statusPageResource{}
	_ resource.ResourceWithConfigure   = &statusPageResource{}
	_ resource.ResourceWithImportState = &statusPageResource{}
)

// NewStatusPageResource is a helper function to simplify the provider implementation.
func NewStatusPageResource() resource.Resource {
	return &statusPageResource{}
}

type statusPageResource struct {
	client *statuspalnext.Client
}

type statusPageResourceModel struct {
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

type notificationSettingsModel struct {
	EmailEnabled   types.Bool `tfsdk:"email_enabled"`
	SlackEnabled   types.Bool `tfsdk:"slack_enabled"`
	RSSFeedEnabled types.Bool `tfsdk:"rss_feed_enabled"`
}

func (r *statusPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *statusPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a StatusPal Next status page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-generated identifier of the status page.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the status page.",
				Required:            true,
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain that addresses the status page (e.g. `acme-status`). " +
					"Changing it forces a new resource, because the API addresses the page by subdomain.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timezone": schema.StringAttribute{
				MarkdownDescription: "Primary timezone used to display incidents (e.g. `America/New_York`).",
				Required:            true,
			},
			"website_url": schema.StringAttribute{
				MarkdownDescription: "URL of the company, project, or service this page reports on.",
				Required:            true,
			},
			"require_authentication": schema.BoolAttribute{
				MarkdownDescription: "When `true`, only signed-in members can view the status page.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"current_status": schema.StringAttribute{
				MarkdownDescription: "Derived overall status: `operational`, `degraded`, or `unavailable`.",
				Computed:            true,
			},
			"notification_settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Subscriber notification channels for this status page.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(defaultNotificationSettings()),
				Attributes: map[string]schema.Attribute{
					"email_enabled": schema.BoolAttribute{
						MarkdownDescription: "Allow subscribers to receive email notifications.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"slack_enabled": schema.BoolAttribute{
						MarkdownDescription: "Allow subscribers to subscribe via Slack.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"rss_feed_enabled": schema.BoolAttribute{
						MarkdownDescription: "Expose an RSS/Atom feed of updates.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the status page was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the status page was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *statusPageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan statusPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateStatusPage(ctx, statusPageRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, mapStatusPageToModel(created))...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state statusPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := r.client.GetStatusPage(ctx, state.Subdomain.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, mapStatusPageToModel(sp))...)
}

func (r *statusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan statusPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateStatusPage(ctx, plan.Subdomain.ValueString(), statusPageRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating status page", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, mapStatusPageToModel(updated))...)
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state statusPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteStatusPage(ctx, state.Subdomain.ValueString()); err != nil {
		if statuspalnext.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting status page", err.Error())
	}
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Status pages are imported by subdomain.
	resource.ImportStatePassthroughID(ctx, path.Root("subdomain"), req, resp)
}

func statusPageRequestBody(plan *statusPageResourceModel) *statuspalnext.StatusPage {
	sp := &statuspalnext.StatusPage{
		Name:                  plan.Name.ValueString(),
		Subdomain:             plan.Subdomain.ValueString(),
		Timezone:              plan.Timezone.ValueString(),
		WebsiteURL:            plan.WebsiteURL.ValueString(),
		RequireAuthentication: statuspalnext.BoolPtr(plan.RequireAuthentication.ValueBool()),
	}

	if plan.NotificationSettings != nil {
		sp.NotificationSettings = &statuspalnext.NotificationSettings{
			EmailEnabled:   statuspalnext.BoolPtr(plan.NotificationSettings.EmailEnabled.ValueBool()),
			SlackEnabled:   statuspalnext.BoolPtr(plan.NotificationSettings.SlackEnabled.ValueBool()),
			RSSFeedEnabled: statuspalnext.BoolPtr(plan.NotificationSettings.RSSFeedEnabled.ValueBool()),
		}
	}

	return sp
}

func mapStatusPageToModel(sp *statuspalnext.StatusPage) statusPageResourceModel {
	model := statusPageResourceModel{
		ID:                    types.StringValue(sp.ID),
		Name:                  types.StringValue(sp.Name),
		Subdomain:             types.StringValue(sp.Subdomain),
		Timezone:              types.StringValue(sp.Timezone),
		WebsiteURL:            types.StringValue(sp.WebsiteURL),
		RequireAuthentication: types.BoolValue(boolValue(sp.RequireAuthentication, false)),
		CurrentStatus:         types.StringValue(sp.CurrentStatus),
		CreatedAt:             types.StringValue(sp.CreatedAt),
		UpdatedAt:             types.StringValue(sp.UpdatedAt),
		NotificationSettings:  &notificationSettingsModel{},
	}

	ns := sp.NotificationSettings
	if ns == nil {
		ns = &statuspalnext.NotificationSettings{}
	}
	model.NotificationSettings.EmailEnabled = types.BoolValue(boolValue(ns.EmailEnabled, true))
	model.NotificationSettings.SlackEnabled = types.BoolValue(boolValue(ns.SlackEnabled, false))
	model.NotificationSettings.RSSFeedEnabled = types.BoolValue(boolValue(ns.RSSFeedEnabled, true))

	return model
}
