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

resource "boundary_credential_library_vault_ldap" "foo-static" {
  name                = "foo"
  description         = "My first static Vault LDAP credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "ldap/static-cred/foo" # change to Vault backend path
}

resource "boundary_credential_library_vault_ldap" "bar-dynamic" {
  name                = "bar"
  description         = "My first dynamic Vault LDAP credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "ldap/creds/bar" # change to Vault backend path
}
