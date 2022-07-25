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
  description = "My first sttatic credential store!"
  scope_id    = boundary_scope.project.id
}
