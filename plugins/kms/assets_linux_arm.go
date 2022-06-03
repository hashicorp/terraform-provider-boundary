package kms_plugin_assets

import (
	"embed"
)

// content is our static kms plugin content.
//go:embed assets/linux/arm
var content embed.FS
