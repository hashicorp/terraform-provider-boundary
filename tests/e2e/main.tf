terraform {
  required_providers {
    boundary = {
      source = "localhost/providers/boundary"
      version = "0.0.1"
    }
  }
}

# Variables for infra
variable "boundary_addr" {
  type    = string
  default = "http://boundary:9200"
}

variable "auth_method_id" {
  type    = string
  default = "ampw_1234567890"
}

variable "auth_method_login_name" {
  type    = string
  default = "admin"
}

variable "auth_method_password" {
  type    = string
  default = "password"
}

# Variables for tests
variable "org_name" {
  type = string
}

resource "boundary_scope" "org" {
  name = var.org_name
  scope_id = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}
