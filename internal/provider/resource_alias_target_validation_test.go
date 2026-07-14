// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestValidateTargetAliasScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scopeId   string
		wantError bool
	}{
		{
			name:      "org scope rejected",
			scopeId:   "o_1234567890",
			wantError: true,
		},
		{
			name:      "global scope allowed",
			scopeId:   globalScopeId,
			wantError: false,
		},
		{
			name:      "project scope allowed",
			scopeId:   "p_1234567890",
			wantError: false,
		},
		{
			name:      "unknown scope type rejected",
			scopeId:   "s_1234567890",
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTargetAliasScope(tt.scopeId)
			if tt.wantError && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
