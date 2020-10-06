resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.global.id
  auto_create_role = true
}
