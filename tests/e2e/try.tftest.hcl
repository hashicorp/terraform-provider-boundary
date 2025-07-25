provider "boundary" {
  addr                   = var.boundary_addr
  auth_method_id         = var.auth_method_id
  auth_method_login_name = var.auth_method_login_name
  auth_method_password   = var.auth_method_password
}

provider "boundary" {
  alias = "recovery"
  addr = var.boundary_addr
  recovery_kms_hcl = <<EOT
    kms "aead" {
      purpose   = "recovery"
      aead_type = "aes-gcm"
      key       = "8fZBjCUfN0TzjEGLQldGY4+iE9AkOvCfjh7+p0GtRBQ="
      key_id    = "global_recovery"
    }
  EOT
}

variables {
  org_name = "mytestorg"
}

run "valid_org_name" {
  command = plan

  assert {
    condition = boundary_scope.org.name == var.org_name
    error_message = "The organization name should match the variable org_name"
  }
}
