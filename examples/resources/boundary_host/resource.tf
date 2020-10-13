resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_scope" "project" {
  name             = "project_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.org.id
  auto_create_role = true
}

resource "boundary_host" "example" {
  name        = "example_host"
  description = "My first host!"
  address     = "10.0.0.1"
  scope_id    = boundary_scope.project.id
}
