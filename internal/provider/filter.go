// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import "fmt"

func FilterWithItemNameMatches(name string) string {
	return fmt.Sprintf("\"/item/name\" matches \"%s\"", name)
}
