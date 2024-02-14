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

resource "boundary_policy_storage" "foo" {
  name                     = "foo"
  description              = "Foo policy"
  scope_id                 = boundary_scope.org.id
  retain_for_days          = 10
  retain_for_overridable   = false
  delete_after_days        = 10
  delete_after_overridable = true
}

resource "boundary_scope_policy_attachment" "foo_attachment" {
  scope_id  = boundary_scope.org.id
  policy_id = boundary_policy_storage.foo.id
}
