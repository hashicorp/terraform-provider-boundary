# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "global scope"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first scope!"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

resource "boundary_credential_store_static" "example" {
  name        = "example_static_credential_store"
  description = "My first static credential store!"
  scope_id    = boundary_scope.project.id
}

resource "boundary_credential_username_password" "example" {
  name                = "example_username_password"
  description         = "My first username password credential!"
  credential_store_id = boundary_credential_store_static.example.id
  username            = "my-username"
  password            = "my-password"
}
