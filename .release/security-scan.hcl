# Copyright IBM Corp. 2020, 2026
# SPDX-License-Identifier: MPL-2.0
#
# Reference: https://github.com/hashicorp/security-scanner/blob/main/CONFIG.md#binary (private repository)

binary {
  secrets {
    all = true
  }
  go_modules   = true
  osv          = true
  oss_index    = false
  nvd          = false

  triage {
    suppress {
      vulnerabilities = [
        # golang.org/x/crypto/openpgp (GO-2026-5932)
        # https://pkg.go.dev/vuln/GO-2026-5932
        #
        # The advisory is specific to openpgp; terraform-provider-boundary does not import this package.
        #
        # $ go mod why golang.org/x/crypto/openpgp
        # # golang.org/x/crypto/openpgp
        # (main module does not need package golang.org/x/crypto/openpgp)
        "GO-2026-5932",
      ]
    }
  }
}
