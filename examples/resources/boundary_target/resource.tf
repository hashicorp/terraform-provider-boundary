resource "boundary_scope" "global" {
  global_scope = true
  scope_id     = "global"
}

resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = boundary_scope.global.id
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first scope!"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

resource "boundary_credential_store_vault" "foo" {
  name        = "vault_store"
  description = "My first Vault credential store!"
  address     = "http://127.0.0.1:8200"      # change to Vault address
  token       = "s.0ufRo6XEGU2jOqnIr7OlFYP5" # change to valid Vault token
  scope_id    = boundary_scope.project.id
}

resource "boundary_credential_library_vault" "foo" {
  name                = "foo"
  description         = "My first Vault credential library!"
  credential_store_id = boundary_credential_store_vault.foo.id
  path                = "my/secret/foo" # change to Vault backend path
  http_method         = "GET"
  credential_type     = "username_password"
}

resource "boundary_host_catalog" "foo" {
  name        = "test"
  description = "test catalog"
  scope_id    = boundary_scope.project.id
  type        = "static"
}

resource "boundary_host" "foo" {
  type            = "static"
  name            = "foo"
  host_catalog_id = boundary_host_catalog.foo.id
  address         = "10.0.0.1"
}

resource "boundary_host" "bar" {
  type            = "static"
  name            = "bar"
  host_catalog_id = boundary_host_catalog.foo.id
  address         = "10.0.0.1"
}

resource "boundary_host_set" "foo" {
  type            = "static"
  name            = "foo"
  host_catalog_id = boundary_host_catalog.foo.id

  host_ids = [
    boundary_host.foo.id,
    boundary_host.bar.id,
  ]
}

resource "boundary_storage_bucket" "aws_example" {
  name            = "My aws storage bucket"
  description     = "My first storage bucket!"
  scope_id        = boundary_scope.org.id
  plugin_name     = "aws"
  bucket_name     = "mybucket"
  attributes_json = jsonencode({ "region" = "us-east-1" })
  secrets_json = jsonencode({
    "access_key_id"     = "aws_access_key_id_value",
    "secret_access_key" = "aws_secret_access_key_value"
  })
  worker_filter = "\"pki\" in \"/tags/type\""
}

resource "boundary_target" "foo" {
  name         = "foo"
  description  = "Foo target"
  type         = "tcp"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  host_source_ids = [
    boundary_host_set.foo.id
  ]
  brokered_credential_source_ids = [
    boundary_credential_library_vault.foo.id
  ]
}

resource "boundary_target" "ssh_foo" {
  name         = "ssh_foo"
  description  = "Ssh target"
  type         = "ssh"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  host_source_ids = [
    boundary_host_set.foo.id
  ]
  injected_application_credential_source_ids = [
    boundary_credential_library_vault.foo.id
  ]
}

resource "boundary_target" "ssh_session_recording_foo" {
  name         = "ssh_foo"
  description  = "Ssh target"
  type         = "ssh"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  host_source_ids = [
    boundary_host_set.foo.id
  ]
  injected_application_credential_source_ids = [
    boundary_credential_library_vault.foo.id
  ]
  enable_session_recording = true
  storage_bucket_id        = boundary_storage_bucket.aws_example
}

resource "boundary_target" "address_foo" {
  name         = "address_foo"
  description  = "Foo target with an address"
  type         = "tcp"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  address      = "127.0.0.1"
}
