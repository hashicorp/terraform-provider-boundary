// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import "fmt"

func FilterWithItemNameMatches(name string) string {
	return fmt.Sprintf("\"/item/name\" matches \"%s\"", name)
}
