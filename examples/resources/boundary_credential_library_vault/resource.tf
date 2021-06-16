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
  address     = "127.0.0.1"
  token       = "s.0ufRo6XEGU2jOqnIr7OlFYP5"
  scope_id    = boundary_scope.project.id
}

resource "boundary_credential_library_vault" "foo" {
  name                = "foo"
  description         = "My first Vault credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  vault_path          = "my/secret/foo"
  http_method         = "GET"
}

resource "boundary_credential_library_vault" "bar" {
  name                = "bar"
  description         = "My second Vault credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  vault_path          = "my/secret/bar"
  http_method         = "POST"
  request_body        = <<EOT
{
  "key": "Value",
}
EOT
}
