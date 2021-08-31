resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_role" "readonly" {
  name          = "readonly"
  description   = "A readonly role"
  principals    = [boundary_user.readonly.id]
  grant_strings = ["id=*;action=read"]
  scope_id      = boundary_scope.org.id
}
