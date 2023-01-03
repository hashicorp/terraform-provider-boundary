# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
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

resource "boundary_host_catalog_static" "example" {
  name        = "My catalog"
  description = "My first host catalog!"
  scope_id    = boundary_scope.project.id
}

resource "boundary_host_static" "example" {
  name            = "example_host"
  description     = "My first host!"
  address         = "10.0.0.1"
  host_catalog_id = boundary_host_catalog_static.example.id
}
