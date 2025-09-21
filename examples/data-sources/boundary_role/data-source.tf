# Role from the global scope
data "boundary_group" "global_role" {
  name = "global_role_one"
}

# Role from an org scope
data "boundary_scope" "org" {
  name     = "org_one"
  scope_id = "global"
}

data "boundary_group" "org_role" {
  name     = "org_role_one"
  scope_id = data.boundary_scope.org.id
}
