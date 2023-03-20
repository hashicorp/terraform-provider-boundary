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

resource "boundary_credential_ssh_private_key" "example" {
  name                   = "example_ssh_private_key"
  description            = "My first ssh private key credential!"
  credential_store_id    = boundary_credential_store_static.example.id
  username               = "my-username"
  private_key            = file("~/.ssh/id_rsa") # change to valid SSH Private Key
  private_key_passphrase = "optional-passphrase" # change to the passphrase of the Private Key if required
}
