resource "boundary_scope" "global" {
  global_scope     = true
  scope_id         = "global"
  auto_create_role = true
}
