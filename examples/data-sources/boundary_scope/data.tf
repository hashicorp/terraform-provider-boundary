data "boundary_scope" "org" {
  name                     = "SecOps"
  scope_id                 = "global"
}

data "boundary_scope" "project" {
	name = "2111"
	scope_id = data.boundary_scope.id
}


