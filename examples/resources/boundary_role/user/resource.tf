resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_user" "foo" {
  name     = "User 1"
  scope_id = boundary_scope.org.id
}

resource "boundary_user" "bar" {
  name     = "User 2"
  scope_id = boundary_scope.org.id
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  principals  = [boundary_user.foo.id, boundary_user.bar.id]
  scope_id    = boundary_scope.org.id
}
