# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_scope" "org" {
  name        = "organization_one"
  description = "My first scope!"
  scope_id    = boundary_scope.global.id
}

resource "boundary_role" "org_admin" {
  scope_id       = boundary_scope.global.id
  grant_scope_id = boundary_scope.org.id
  grant_strings  = ["id=*;type=*;actions=*"]
  principal_ids  = ["u_auth"]
}
