// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"terraform-provider-statuspal-next/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run the docs generation tool. See its repository for how it works and how
// docs can be customized: https://github.com/hashicorp/terraform-plugin-docs
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name statuspal-next

// version is set by the goreleaser configuration to the released version, and
// is "dev" when the provider is built and run locally.
var version string = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/statuspal/statuspal-next",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
