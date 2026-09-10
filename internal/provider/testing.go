// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
)

type config struct {
	BoundaryAddr        string `envconfig:"BOUNDARY_ADDR" required:"true"`
	BoundaryAuthToken   string `envconfig:"BOUNDARY_AUTH_TOKEN" required:"true"`
	BoundaryRecoveryKey string `envconfig:"BOUNDARY_DEV_RECOVERY_KEY"`
}

func loadTestConfig() (*config, error) {
	var c config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// requireRecoveryKey returns cfg.BoundaryRecoveryKey or calls t.Skip when the
// env var is not set. Call this only in tests that actually use the recovery KMS.
func requireRecoveryKey(t *testing.T, cfg *config) string {
	t.Helper()
	if cfg.BoundaryRecoveryKey == "" {
		t.Fatal("BOUNDARY_DEV_RECOVERY_KEY not set")
	}
	return cfg.BoundaryRecoveryKey
}
