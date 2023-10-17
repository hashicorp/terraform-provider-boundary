# Retrieve the ID of a Boundary project
data "boundary_scope" "org" {
  name            = "SecOps"
  parent_scope_id = "global"
}

data "boundary_scope" "project" {
  name            = "2111"
  parent_scope_id = data.boundary_scope.id
}
