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

resource "boundary_credential_library_vault" "foo" {
  name                = "foo"
  description         = "My first Vault credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "my/secret/foo" # change to Vault backend path
  http_method         = "GET"
}

resource "boundary_credential_library_vault" "bar" {
  name                = "bar"
  description         = "My second Vault credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "my/secret/bar" # change to Vault backend path
  http_method         = "POST"
  http_request_body   = <<EOT
{
  "key": "Value",
}
EOT
}

resource "boundary_credential_library_vault" "baz" {
  name                = "baz"
  description         = "vault username password credential with mapping overrides"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "my/secret/baz" # change to Vault backend path
  http_method         = "GET"
  credential_type     = "username_password"
  credential_mapping_overrides = {
    password_attribute = "alternative_password_label"
    username_attribute = "alternative_username_label"
  }
}

resource "boundary_credential_library_vault" "quz" {
  name                = "quz"
  description         = "vault ssh private key credential with mapping overrides"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "my/secret/quz" # change to Vault backend path
  http_method         = "GET"
  credential_type     = "ssh_private_key"
  credential_mapping_overrides = {
    private_key_attribute            = "alternative_key_label"
    private_key_passphrase_attribute = "alternative_passphrase_label"
    username_attribute               = "alternative_username_label"
  }
}
