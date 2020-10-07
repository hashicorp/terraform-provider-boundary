resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_auth_method" "password" {
  scope_id = boundary_scope.org.id
  type     = "password"
}

resource "boundary_account" "jeff" {
  auth_method_id = boundary_auth_method.password.id
  type           = "password"
  login_name     = "jeff"
  password       = "$uper$ecure"
}

resource "boundary_user" "jeff" {
  name        = "jeff"
  description = "Jeff's user resource"
  account_ids = [boundary_account.jeff.id]
  scope_id    = boundary_scope.org.id
}
