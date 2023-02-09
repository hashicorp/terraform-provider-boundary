resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  scope_id    = boundary_scope.org.id
}
