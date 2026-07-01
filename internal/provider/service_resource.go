// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// containerStatusAttrTypes describes one entry of a service's container_statuses.
var containerStatusAttrTypes = map[string]attr.Type{
	"container_slug": types.StringType,
	"status":         types.StringType,
}

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

// NewServiceResource is a helper function to simplify the provider implementation.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	client *statuspalnext.Client
}

type serviceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	StatusPageSubdomain types.String `tfsdk:"status_page_subdomain"`
	Name                types.String `tfsdk:"name"`
	Slug                types.String `tfsdk:"slug"`
	Description         types.String `tfsdk:"description"`
	Order               types.Int64  `tfsdk:"order"`
	Status              types.String `tfsdk:"status"`
	ContainerStatuses   types.List   `tfsdk:"container_statuses"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service displayed on a StatusPal Next status page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-generated identifier of the service.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page the service belongs to. " +
					"Changing it forces a new resource.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the service.",
				Required:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "URL-safe identifier, unique per status page. If omitted it is " +
					"auto-generated from `name` (collisions get a numeric suffix).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description shown beside the service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"order": schema.Int64Attribute{
				MarkdownDescription: "Position in the status page's service list (1-indexed).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Computed worst-case live status across all containers: " +
					"`ok`, `minor`, `major`, or `maintenance`. Live status is operational state, " +
					"managed via notices in the app, not by Terraform.",
				Computed: true,
			},
			"container_statuses": schema.ListNestedAttribute{
				MarkdownDescription: "Per-container live status for this service (read-only).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"container_slug": schema.StringAttribute{
							MarkdownDescription: "Slug of the container.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Live status in that container.",
							Computed:            true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the service was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the service was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := plan.StatusPageSubdomain.ValueString()
	created, err := r.client.CreateService(ctx, subdomain, serviceRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	model, diags := mapServiceToModel(subdomain, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := state.StatusPageSubdomain.ValueString()
	svc, err := r.client.GetService(ctx, subdomain, state.Slug.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}

	model, diags := mapServiceToModel(subdomain, svc)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := plan.StatusPageSubdomain.ValueString()
	// The path uses the current (state) slug; the body may carry a new slug.
	updated, err := r.client.UpdateService(ctx, subdomain, state.Slug.ValueString(), serviceRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating service", err.Error())
		return
	}

	model, diags := mapServiceToModel(subdomain, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteService(ctx, state.StatusPageSubdomain.ValueString(), state.Slug.ValueString())
	if err != nil && !statuspalnext.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "<status_page_subdomain>/<service_slug>".
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format \"status_page_subdomain/service_slug\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_subdomain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), parts[1])...)
}

func serviceRequestBody(plan *serviceResourceModel) *statuspalnext.Service {
	svc := &statuspalnext.Service{
		Name: plan.Name.ValueString(),
	}

	if !plan.Slug.IsUnknown() && !plan.Slug.IsNull() && plan.Slug.ValueString() != "" {
		svc.Slug = plan.Slug.ValueString()
	}
	if !plan.Description.IsUnknown() && !plan.Description.IsNull() {
		svc.Description = statuspalnext.StringPtr(plan.Description.ValueString())
	}
	if !plan.Order.IsUnknown() && !plan.Order.IsNull() {
		svc.Order = statuspalnext.Int64Ptr(plan.Order.ValueInt64())
	}

	return svc
}

func mapServiceToModel(subdomain string, svc *statuspalnext.Service) (serviceResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := serviceResourceModel{
		ID:                  types.StringValue(svc.ID),
		StatusPageSubdomain: types.StringValue(subdomain),
		Name:                types.StringValue(svc.Name),
		Slug:                types.StringValue(svc.Slug),
		Order:               types.Int64Value(int64Value(svc.Order)),
		Status:              types.StringValue(svc.Status),
		CreatedAt:           types.StringValue(svc.CreatedAt),
		UpdatedAt:           types.StringValue(svc.UpdatedAt),
	}

	if svc.Description == nil {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(*svc.Description)
	}

	elems := make([]attr.Value, 0, len(svc.ContainerStatuses))
	for _, cs := range svc.ContainerStatuses {
		obj, objDiags := types.ObjectValue(containerStatusAttrTypes, map[string]attr.Value{
			"container_slug": types.StringValue(cs.ContainerSlug),
			"status":         types.StringValue(cs.Status),
		})
		diags.Append(objDiags...)
		elems = append(elems, obj)
	}
	list, listDiags := types.ListValue(types.ObjectType{AttrTypes: containerStatusAttrTypes}, elems)
	diags.Append(listDiags...)
	model.ContainerStatuses = list

	return model, diags
}

func int64Value(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
