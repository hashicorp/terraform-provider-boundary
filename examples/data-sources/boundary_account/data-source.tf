# Retrieve the ID of a Boundary account
data "boundary_account" "admin" {
  name           = "admin"
  auth_method_id = "ampw_1234567890"
}
