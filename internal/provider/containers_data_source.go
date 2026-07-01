// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type containerDataModel struct {
	ID        types.String `tfsdk:"id"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	Services  types.List   `tfsdk:"services"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func containerToDataModel(ct *statuspalnext.Container) (containerDataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := containerDataModel{
		ID:        types.StringValue(ct.ID),
		Slug:      types.StringValue(ct.Slug),
		Name:      types.StringValue(ct.Name),
		CreatedAt: types.StringValue(ct.CreatedAt),
		UpdatedAt: types.StringValue(ct.UpdatedAt),
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

var (
	_ datasource.DataSource              = &containersDataSource{}
	_ datasource.DataSourceWithConfigure = &containersDataSource{}
)

// NewContainersDataSource is a helper function to simplify the provider implementation.
func NewContainersDataSource() datasource.DataSource {
	return &containersDataSource{}
}

type containersDataSource struct {
	client *statuspalnext.Client
}

type containersDataSourceModel struct {
	StatusPageSubdomain types.String         `tfsdk:"status_page_subdomain"`
	Containers          []containerDataModel `tfsdk:"containers"`
}

func (d *containersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_containers"
}

func (d *containersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the containers (regions) on a status page.",
		Attributes: map[string]schema.Attribute{
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page whose containers to list.",
				Required:            true,
			},
			"containers": schema.ListNestedAttribute{
				MarkdownDescription: "Containers on the status page.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"slug": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"services": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"slug":   schema.StringAttribute{Computed: true},
									"name":   schema.StringAttribute{Computed: true},
									"status": schema.StringAttribute{Computed: true},
								},
							},
						},
						"created_at": schema.StringAttribute{Computed: true},
						"updated_at": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *containersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *containersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config containersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	containers, err := d.client.ListContainers(ctx, config.StatusPageSubdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing containers", err.Error())
		return
	}

	config.Containers = make([]containerDataModel, 0, len(containers))
	for i := range containers {
		model, diags := containerToDataModel(&containers[i])
		resp.Diagnostics.Append(diags...)
		config.Containers = append(config.Containers, model)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
