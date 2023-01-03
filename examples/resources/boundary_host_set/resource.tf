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
  type     = "static"
  scope_id = boundary_scope.project.id
}

resource "boundary_host" "first" {
  type            = "static"
  name            = "host_1"
  description     = "My first host!"
  address         = "10.0.0.1"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host" "second" {
  type            = "static"
  name            = "host_2"
  description     = "My second host!"
  address         = "10.0.0.2"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host_set" "web" {
  host_catalog_id = boundary_host_catalog.static.id
  type            = "static"
  host_ids = [
    boundary_host.first.id,
    boundary_host.second.id,
  ]
}
