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

resource "boundary_credential_store_vault" "foo" {
  name        = "foo"
  description = "My first Vault credential store!"
  address     = "http://127.0.0.1:8200"      # change to Vault address
  token       = "s.0ufRo6XEGU2jOqnIr7OlFYP5" # change to valid Vault token
  scope_id    = boundary_scope.project.id
}

resource "boundary_credential_library_vault_ssh_certificate" "foo" {
  name                = "foo"
  description         = "My first Vault SSH certificate credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "ssh/sign/foo" # change to correct Vault endpoint and role
  username            = "foo"          # change to valid username
}

resource "boundary_credential_library_vault_ssh_certificate" "bar" {
  name                = "bar"
  description         = "My second Vault SSH certificate credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "ssh/sign/foo" # change to correct Vault endpoint and role
  username            = "foo"
  key_type            = "ecdsa"
  key_bits            = 384

  extensions = {
    permit-pty = ""
  }
}

resource "boundary_credential_library_vault_ssh_certificate" "baz" {
  name                = "baz"
  description         = "vault "
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "ssh/issue/foo" # change to correct Vault endpoint and role
  username            = "foo"
  key_type            = "rsa"
  key_bits            = 4096

  extensions = {
    permit-pty            = ""
    permit-X11-forwarding = ""
  }

  critical_options = {
    force-command = "/bin/some_script"
  }
}
