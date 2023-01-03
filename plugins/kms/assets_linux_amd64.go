// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kms_plugin_assets

import (
	"embed"
)

// content is our static kms plugin content.
//
//go:embed assets/linux/amd64
var content embed.FS
