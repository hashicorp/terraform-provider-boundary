# Retrieve a user from the global scope
data "boundary_user" "global_scope_admin" {
  name = "admin"
}

# User from a org scope
data "boundary_scope" "org" {
  name     = "my-org"
  scope_id = "global"
}

data "boundary_user" "org_user" {
  name     = "username"
  scope_id = data.boundary_scope.org.id
}
