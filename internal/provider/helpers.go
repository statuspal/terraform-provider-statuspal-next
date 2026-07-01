// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	statuspalnext "terraform-provider-statuspal-next/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// clientFromProviderData performs the standard type assertion used by every
// resource and data source Configure method. It returns nil (without adding an
// error) when providerData is nil, because Terraform calls Configure with nil
// data before the provider itself has been configured.
func clientFromProviderData(providerData any, diags *diag.Diagnostics) *statuspalnext.Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*statuspalnext.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *statuspalnext.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

// boolValue dereferences an optional bool pointer from an API response, falling
// back to def when the field was absent/null.
func boolValue(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// optionalString maps a nullable string pointer from an API response to a
// types.String, yielding null when the pointer is nil or empty.
func optionalString(p *string) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
