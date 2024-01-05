# Retrieve an auth method from the global scope
data "boundary_auth_method" "auth_method" {
  name = "password_auth_method"
}

# Auth method from a org scope
data "boundary_scope" "org" {
  name     = "my-org"
  scope_id = "global"
}

data "boundary_auth_method" "auth_method" {
  name     = "password_auth_method"
  scope_id = data.boundary_scope.org.id
}
