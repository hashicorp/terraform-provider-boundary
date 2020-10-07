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

resource "boundary_host_catalog" "static" {
  scope_id = boundary_scope.project.id
}

resource "boundary_host" "1" {
  name            = "host_1"
  description     = "My first host!"
  address         = "10.0.0.1"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host" "2" {
  name            = "host_2"
  description     = "My second host!"
  address         = "10.0.0.2"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host_set" "web" {
  host_catalog_id = boundary_host_catalog.static.id
  host_ids = [
    boundary_host.1.id,
    boundary_host.2.id,
  ]
}
