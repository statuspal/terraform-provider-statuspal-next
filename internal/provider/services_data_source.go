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

type serviceDataModel struct {
	ID                types.String `tfsdk:"id"`
	Slug              types.String `tfsdk:"slug"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Order             types.Int64  `tfsdk:"order"`
	Status            types.String `tfsdk:"status"`
	ContainerStatuses types.List   `tfsdk:"container_statuses"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func serviceToDataModel(svc *statuspalnext.Service) (serviceDataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := serviceDataModel{
		ID:        types.StringValue(svc.ID),
		Slug:      types.StringValue(svc.Slug),
		Name:      types.StringValue(svc.Name),
		Order:     types.Int64Value(int64Value(svc.Order)),
		Status:    types.StringValue(svc.Status),
		CreatedAt: types.StringValue(svc.CreatedAt),
		UpdatedAt: types.StringValue(svc.UpdatedAt),
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

var (
	_ datasource.DataSource              = &servicesDataSource{}
	_ datasource.DataSourceWithConfigure = &servicesDataSource{}
)

// NewServicesDataSource is a helper function to simplify the provider implementation.
func NewServicesDataSource() datasource.DataSource {
	return &servicesDataSource{}
}

type servicesDataSource struct {
	client *statuspalnext.Client
}

type servicesDataSourceModel struct {
	StatusPageSubdomain types.String       `tfsdk:"status_page_subdomain"`
	Services            []serviceDataModel `tfsdk:"services"`
}

func (d *servicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *servicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the services on a status page.",
		Attributes: map[string]schema.Attribute{
			"status_page_subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the status page whose services to list.",
				Required:            true,
			},
			"services": schema.ListNestedAttribute{
				MarkdownDescription: "Services on the status page.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"slug":        schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"order":       schema.Int64Attribute{Computed: true},
						"status":      schema.StringAttribute{Computed: true},
						"container_statuses": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"container_slug": schema.StringAttribute{Computed: true},
									"status":         schema.StringAttribute{Computed: true},
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

func (d *servicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *servicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config servicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	services, err := d.client.ListServices(ctx, config.StatusPageSubdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing services", err.Error())
		return
	}

	config.Services = make([]serviceDataModel, 0, len(services))
	for i := range services {
		model, diags := serviceToDataModel(&services[i])
		resp.Diagnostics.Append(diags...)
		config.Services = append(config.Services, model)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
