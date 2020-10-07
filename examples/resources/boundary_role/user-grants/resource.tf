resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
  scope_id    = boundary_scope.org.id
}
