// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "testing"

func TestAliasSuffixFromScopeResponseMap(t *testing.T) {
	t.Parallel()

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		_, err := aliasSuffixFromScopeResponseMap(nil)
		if err == nil {
			t.Fatal("expected error for nil map")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()

		got, err := aliasSuffixFromScopeResponseMap(map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty alias suffix, got %q", got)
		}
	})

	t.Run("nil value", func(t *testing.T) {
		t.Parallel()

		got, err := aliasSuffixFromScopeResponseMap(map[string]interface{}{aliasSuffixKey: nil})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty alias suffix, got %q", got)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		t.Parallel()

		_, err := aliasSuffixFromScopeResponseMap(map[string]interface{}{aliasSuffixKey: 123})
		if err == nil {
			t.Fatal("expected error for non-string alias suffix")
		}
	})

	t.Run("string value", func(t *testing.T) {
		t.Parallel()

		got, err := aliasSuffixFromScopeResponseMap(map[string]interface{}{aliasSuffixKey: "example.boundary"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "example.boundary" {
			t.Fatalf("unexpected alias suffix: %q", got)
		}
	})
}
