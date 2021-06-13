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

resource "boundary_credential_store_vault" "example" {
  name        = "vault_store"
  description = "My first Vault credential store!"
  address     = "http://localhost:55001"
  token       = "s.0ufRo6XEGU2jOqnIr7OlFYP5"
  scope_id    = boundary_scope.project.id
}

resource "boundary_credential_library_vault" "example" {
  name                = "vault_library"
  description         = "My first Vault credential library!"
  credential_store_id = boundary_credential_store_vault.example.id
  vault_path          = "database/creds/opened"
  http_method         = "GET"
}
