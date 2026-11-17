# Local development example using a locally built provider and Boundary dev server.
terraform {
  required_providers {
    boundary = {
      source  = "localhost/providers/boundary"
      version = "0.0.1"
    }
  }
}

variable "boundary_password" {
  description = "Password for the local Boundary dev admin. Set with TF_VAR_boundary_password."
  type        = string
  sensitive   = true
}

provider "boundary" {
  addr                   = var.boundary_addr
  auth_method_id         = "ampw_1234567890"
  auth_method_login_name = "admin"
  auth_method_password   = var.boundary_password
}

variable "boundary_addr" {
  description = "Address of the Boundary controller to test against."
  type        = string
  default     = "http://127.0.0.1:9200"
}

resource "boundary_managed_group" "testNoStrictFiltering" {
  name           = "strict-filter-test"
  description    = "Testing with the strict-filter setting omitted"
  auth_method_id = "amoidc_1234567890"
  filter         = "\"/token/email\" == \"alice@example.test\""

  # Omission uses Boundary's creation default (strict evaluation on) for new groups
  # and preserves the server value for existing groups. Uncomment to use legacy behavior.
  # disable_strict_filter_evaluation = true
}
