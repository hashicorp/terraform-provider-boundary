// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import "github.com/hashicorp/boundary/api/scopes"

func flattenScopeInfo(scope *scopes.ScopeInfo) []interface{} {
	if scope == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if v := scope.Id; v != "" {
		m[IDKey] = v
	}
	if v := scope.Type; v != "" {
		m[TypeKey] = v
	}
	if v := scope.Description; v != "" {
		m[DescriptionKey] = v
	}
	if v := scope.ParentScopeId; v != "" {
		m[ParentScopeIdKey] = v
	}
	if v := scope.Name; v != "" {
		m[NameKey] = v
	}

	return []interface{}{m}
}
