resource "boundary_scope" "global" {
  global_scope = true
  scope_id     = "global"
}

resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = boundary_scope.global.id
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first project"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

resource "boundary_scope_alias_suffix" "org_suffix" {
  scope_id     = boundary_scope.org.id
  alias_suffix = "org"
}

resource "boundary_scope_alias_suffix" "project_suffix" {
  scope_id     = boundary_scope.project.id
  alias_suffix = "projectone"
}
