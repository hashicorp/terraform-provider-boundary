# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_scope" "global" {
  global_scope = true
  scope_id     = "global"
}
