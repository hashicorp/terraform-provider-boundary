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

resource "boundary_host_catalog" "static" {
  name        = "My catalog"
  description = "My first host catalog!"
  type        = "static"
  scope_id    = boundary_scope.project.id
}

resource "boundary_host" "example" {
  type            = "static"
  name            = "example_host"
  description     = "My first host!"
  address         = "10.0.0.1"
  host_catalog_id = boundary_host_catalog.static.id
}
