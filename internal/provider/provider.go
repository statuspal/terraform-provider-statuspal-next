// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strings"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	envAPIKey   = "STATUSPAL_NEXT_API_KEY"
	envEndpoint = "STATUSPAL_NEXT_ENDPOINT"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &statuspalNextProvider{}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &statuspalNextProvider{version: version}
	}
}

// statuspalNextProvider is the provider implementation.
type statuspalNextProvider struct {
	// version is set to the provider version on release, and "dev" when built
	// and run locally.
	version string
}

// statuspalNextProviderModel maps provider schema data to a Go type.
type statuspalNextProviderModel struct {
	APIKey   types.String `tfsdk:"api_key"`
	Endpoint types.String `tfsdk:"endpoint"`
}

// Metadata returns the provider type name.
func (p *statuspalNextProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "statuspal-next"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *statuspalNextProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [StatusPal Next](https://www.statuspal.io) status pages, services, " +
			"containers (regions), outgoing webhooks, monitoring checks, and incident automations as code " +
			"via the Management API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Organization API key (a `sk_…` token sent as a Bearer credential). " +
					"May also be provided via the `" + envAPIKey + "` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Management API. Defaults to `" + statuspalnext.DefaultEndpoint +
					"`. May also be provided via the `" + envEndpoint + "` environment variable. Useful for " +
					"pointing at a self-hosted or local development instance (e.g. `http://localhost:7070/api/v1`).",
				Optional: true,
			},
		},
	}
}

// Configure prepares a StatusPal Next API client for data sources and resources.
func (p *statuspalNextProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring StatusPal Next client")

	var config statuspalNextProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values must be known before the client can be built.
	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown StatusPal Next API Key",
			"The provider cannot create the API client because there is an unknown value for the API key. "+
				"Apply the source of the value first, set it statically in configuration, or use the "+
				envAPIKey+" environment variable.",
		)
	}
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown StatusPal Next Endpoint",
			"The provider cannot create the API client because there is an unknown value for the endpoint.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve values: environment variable first, overridden by explicit config.
	apiKey := os.Getenv(envAPIKey)
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	endpoint := os.Getenv(envEndpoint)
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	endpoint = strings.TrimSpace(endpoint)

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing StatusPal Next API Key",
			"The provider requires an API key. Set the `api_key` attribute or the "+envAPIKey+
				" environment variable. If either is already set, ensure the value is not empty.",
		)
		return
	}

	ctx = tflog.SetField(ctx, "statuspal_next_endpoint", endpoint)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "api_key")
	tflog.Debug(ctx, "Creating StatusPal Next client")

	client, err := statuspalnext.NewClient(&apiKey, &endpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create StatusPal Next API Client",
			"An unexpected error occurred when creating the API client: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured StatusPal Next client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *statuspalNextProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStatusPagesDataSource,
		NewStatusPageDataSource,
		NewServicesDataSource,
		NewContainersDataSource,
		NewOutgoingWebhooksDataSource,
		NewMonitoringChecksDataSource,
		NewAutomationsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *statuspalNextProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewStatusPageResource,
		NewServiceResource,
		NewContainerResource,
		NewOutgoingWebhookResource,
		NewMonitoringCheckResource,
		NewAutomationResource,
	}
}
