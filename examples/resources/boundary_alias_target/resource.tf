resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "global scope"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first scope!"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

resource "boundary_host_catalog_static" "foo" {
  name        = "test"
  description = "test catalog"
  scope_id    = boundary_scope.project.id
}

resource "boundary_host_static" "foo" {
  name            = "foo"
  host_catalog_id = boundary_host_catalog_static.foo.id
  address         = "10.0.0.1"
}

resource "boundary_host_static" "bar" {
  name            = "bar"
  host_catalog_id = boundary_host_catalog_static.foo.id
  address         = "127.0.0.1"
}

resource "boundary_host_set_static" "foo" {
  name            = "foo"
  host_catalog_id = boundary_host_catalog_static.foo.id

  host_ids = [
    boundary_host_static.foo.id,
    boundary_host_static.bar.id,
  ]
}

resource "boundary_target" "foo" {
  name         = "foo"
  description  = "Foo target"
  type         = "tcp"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  host_source_ids = [
    boundary_host_set_static.foo.id,
  ]
}

resource "boundary_alias_target" "example_alias_target" {
  name                      = "example_alias_target"
  description               = "Example alias to target foo using host boundary_host_static.bar"
  scope_id                  = "global"
  value                     = "example.bar.foo.boundary"
  destination_id            = boundary_target.foo.id
  authorize_session_host_id = boundary_host_static.bar.id
}