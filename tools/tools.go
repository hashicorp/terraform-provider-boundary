// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build tools
// +build tools

package tools

//go:generate go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

import (
	// docs generator
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
