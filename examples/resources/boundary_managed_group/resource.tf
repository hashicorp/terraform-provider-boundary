variable "oidc_auth_method_id" {
  description = "The ID of an existing Boundary OIDC auth method."
  type        = string
}

resource "boundary_managed_group" "example" {
  name                             = "example-oidc-group"
  description                      = "OIDC managed group for Alice"
  auth_method_id                   = var.oidc_auth_method_id
  filter                           = "\"/token/email\" == \"alice@example.test\""
  disable_strict_filter_evaluation = false
}
