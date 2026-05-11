// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package kms_plugin_assets

import (
	"embed"
)

// content is our static kms plugin content.
//
//go:embed assets/darwin/amd64
var content embed.FS
