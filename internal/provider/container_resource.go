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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// containerServiceAttrTypes describes one entry of a container's services list.
var containerServiceAttrTypes = map[string]attr.Type{
	"slug":   types.StringType,
	"name":   types.StringType,
	"status": types.StringType,
}

var (
	_ resource.Resource                = &containerResource{}
	_ resource.ResourceWithConfigure   = &containerResource{}
	_ resource.ResourceWithImportState = &containerResource{}
)

// NewContainerResource is a helper function to simplify the provider implementation.
func NewContainerResource() resource.Resource {
	return &containerResource{}
}

type containerResource struct {
	client *statuspalnext.Client
}

type containerResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	StatusPageSubdomain types.String `tfsdk:"status_page_subdomain"`
	Name                types.String `tfsdk:"name"`
	Slug                types.String `tfsdk:"slug"`
	Services            types.List   `tfsdk:"services"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a container (service group / region) on a StatusPal Next status page. " +
			"Service membership is managed automatically: every existing service is assigned to a new " +
			"container (full N×M grid).\n\n" +
			"~> **Note:** Every status page is created with a `Default` container automatically. To manage " +
			"that container with Terraform, import it (`terraform import ... <subdomain>/default`) rather " +
			"than declaring a new one with the same name, which would create a second container.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-generated identifier of the container.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page the container belongs to. " +
					"Changing it forces a new resource.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the container (e.g. `EU Region`).",
				Required:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "URL-safe identifier, unique per status page. If omitted it is " +
					"auto-generated from `name` (collisions get a numeric suffix).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"services": schema.ListNestedAttribute{
				MarkdownDescription: "Services assigned to this container, with their live status (read-only).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug":   schema.StringAttribute{MarkdownDescription: "Service slug.", Computed: true},
						"name":   schema.StringAttribute{MarkdownDescription: "Service name.", Computed: true},
						"status": schema.StringAttribute{MarkdownDescription: "Live status of the service in this container.", Computed: true},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the container was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp at which the container was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *containerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := plan.StatusPageSubdomain.ValueString()
	created, err := r.client.CreateContainer(ctx, subdomain, containerRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating container", err.Error())
		return
	}

	model, diags := mapContainerToModel(subdomain, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := state.StatusPageSubdomain.ValueString()
	ct, err := r.client.GetContainer(ctx, subdomain, state.Slug.ValueString())
	if err != nil {
		if statuspalnext.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading container", err.Error())
		return
	}

	model, diags := mapContainerToModel(subdomain, ct)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state containerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := plan.StatusPageSubdomain.ValueString()
	updated, err := r.client.UpdateContainer(ctx, subdomain, state.Slug.ValueString(), containerRequestBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating container", err.Error())
		return
	}

	model, diags := mapContainerToModel(subdomain, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteContainer(ctx, state.StatusPageSubdomain.ValueString(), state.Slug.ValueString())
	if err != nil && !statuspalnext.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting container", err.Error())
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "<status_page_subdomain>/<container_slug>".
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format \"status_page_subdomain/container_slug\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_subdomain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), parts[1])...)
}

func containerRequestBody(plan *containerResourceModel) *statuspalnext.Container {
	ct := &statuspalnext.Container{
		Name: plan.Name.ValueString(),
	}
	if !plan.Slug.IsUnknown() && !plan.Slug.IsNull() && plan.Slug.ValueString() != "" {
		ct.Slug = plan.Slug.ValueString()
	}
	return ct
}

func mapContainerToModel(subdomain string, ct *statuspalnext.Container) (containerResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := containerResourceModel{
		ID:                  types.StringValue(ct.ID),
		StatusPageSubdomain: types.StringValue(subdomain),
		Name:                types.StringValue(ct.Name),
		Slug:                types.StringValue(ct.Slug),
		CreatedAt:           types.StringValue(ct.CreatedAt),
		UpdatedAt:           types.StringValue(ct.UpdatedAt),
	}

	elems := make([]attr.Value, 0, len(ct.Services))
	for _, svc := range ct.Services {
		obj, objDiags := types.ObjectValue(containerServiceAttrTypes, map[string]attr.Value{
			"slug":   types.StringValue(svc.Slug),
			"name":   types.StringValue(svc.Name),
			"status": types.StringValue(svc.Status),
		})
		diags.Append(objDiags...)
		elems = append(elems, obj)
	}
	list, listDiags := types.ListValue(types.ObjectType{AttrTypes: containerServiceAttrTypes}, elems)
	diags.Append(listDiags...)
	model.Services = list

	return model, diags
}
