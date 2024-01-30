resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = boundary_scope.global.id
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_storage_bucket" "aws_static_credentials_example" {
  name            = "My aws storage bucket with static credentials"
  description     = "My first storage bucket!"
  scope_id        = boundary_scope.org.id
  plugin_name     = "aws"
  bucket_name     = "mybucket"
  attributes_json = jsonencode({ "region" = "us-east-1" })

  # recommended to pass in aws secrets using a file() or using environment variables
  # the secrets below must be generated in aws by creating a aws iam user with programmatic access
  secrets_json = jsonencode({
    "access_key_id"     = "aws_access_key_id_value",
    "secret_access_key" = "aws_secret_access_key_value"
  })
  worker_filter = "\"pki\" in \"/tags/type\""
}

resource "boundary_storage_bucket" "aws_dynamic_credentials_example" {
  name            = "My aws storage bucket with dynamic credentials"
  description     = "My first storage bucket!"
  scope_id        = boundary_scope.org.id
  plugin_name     = "aws"
  bucket_name     = "mybucket"
  
  # the role_arn value should be the same arn used as the instance profile that is attached to the ec2 instance
  # https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html
  attributes_json = jsonencode({
    "region" = "us-east-1" 
    "role_arn" = "arn:aws:iam::123456789012:role/S3Access"
  })
  worker_filter = "\"pki\" in \"/tags/type\""
}