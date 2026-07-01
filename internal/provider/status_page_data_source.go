// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &statusPageDataSource{}
	_ datasource.DataSourceWithConfigure = &statusPageDataSource{}
)

// NewStatusPageDataSource is a helper function to simplify the provider implementation.
func NewStatusPageDataSource() datasource.DataSource {
	return &statusPageDataSource{}
}

type statusPageDataSource struct {
	client *statuspalnext.Client
}

func (d *statusPageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (d *statusPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single status page by subdomain.",
		Attributes:          statusPageDataSourceAttributes(true),
	}
}

func (d *statusPageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *statusPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config statusPageDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sp, err := d.client.GetStatusPage(ctx, config.Subdomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, statusPageToDataModel(sp))...)
}
