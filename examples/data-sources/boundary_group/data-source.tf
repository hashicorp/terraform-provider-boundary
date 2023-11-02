# Retrieve a user from the global scope
data "boundary_group" "global_group" {
  name = "admin"
}

# User from an org scope
data "boundary_scope" "org" {
  name     = "org"
  scope_id = "global"
}

data "boundary_group" "org_group" {
  name     = "username"
  scope_id = data.boundary_scope.org.id
}
