resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_auth_method_password" "password" {
  scope_id = boundary_scope.org.id
}

resource "boundary_auth_method_password" "password_is_primary" {
  scope_id = boundary_scope.org.id
  is_primary_for_scope = true
}
