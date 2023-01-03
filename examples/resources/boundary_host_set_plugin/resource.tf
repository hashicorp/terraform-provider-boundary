# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first scope!"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

# For more information about the aws plugin, please visit here:
# https://github.com/hashicorp/boundary-plugin-host-aws
#
# For more information about aws users, please visit here:
# https://learn.hashicorp.com/tutorials/boundary/aws-host-catalogs?in=boundary/oss-access-management#configure-terraform-and-iam-user-privileges
resource "boundary_host_catalog_plugin" "aws_example" {
  name            = "My aws catalog"
  description     = "My first host catalog!"
  scope_id        = boundary_scope.project.id
  plugin_name     = "aws"
  attributes_json = jsonencode({ "region" = "us-east-1" })

  # recommended to pass in aws secrets using a file() or using environment variables
  # the secrets below must be generated in aws by creating a aws iam user with programmatic access
  secrets_json = jsonencode({
    "access_key_id"     = "aws_access_key_id_value",
    "secret_access_key" = "aws_secret_access_key_value"
  })
}

resource "boundary_host_set_plugin" "web" {
  name            = "My web host set plugin"
  host_catalog_id = boundary_host_catalog_plugin.aws_exmaple.id
  attributes_json = jsonencode({ "filters" = "tag:service-type=web" })
}

resource "boundary_host_set_plugin" "foobar" {
  name                = "My foobar host set plugin"
  host_catalog_id     = boundary_host_catalog_plugin.aws_exmaple.id
  preferred_endpoints = ["cidr:54.0.0.0/8"]
  attributes_json = jsonencode({
    "filters" = "tag-key=foo",
    "filters" = "tag-key=bar"
  })
}

resource "boundary_host_set_plugin" "launch" {
  name                  = "My launch host set plugin"
  host_catalog_id       = boundary_host_catalog_plugin.aws_exmaple.id
  sync_interval_seconds = 60
  attributes_json = jsonencode({
    "filters" = "tag:development=prod,dev",
    "filters" = "launch-time=2022-01-04T*"
  })
}

# For more information about the azure plugin, please visit here:
# https://github.com/hashicorp/boundary-plugin-host-azure
#
# For more information about azure ad applications, please visit here:
# https://learn.hashicorp.com/tutorials/boundary/azure-host-catalogs#register-a-new-azure-ad-application-1
resource "boundary_host_catalog_plugin" "azure_example" {
  name        = "My azure catalog"
  description = "My second host catalog!"
  scope_id    = boundary_scope.project.id
  plugin_name = "azure"

  # the attributes below must be generated in azure by creating an ad application
  attributes_json = jsonencode({
    "disable_credential_rotation" = "true",
    "tenant_id"                   = "ARM_TENANT_ID",
    "subscription_id"             = "ARM_SUBSCRIPTION_ID",
    "client_id"                   = "ARM_CLIENT_ID"
  })

  # recommended to pass in aws secrets using a file() or using environment variables
  # the secrets below must be generated in azure by creating an ad application
  secrets_json = jsonencode({
    "secret_value" = "ARM_CLIENT_SECRET"
  })
}

resource "boundary_host_set_plugin" "database" {
  name            = "My database host set plugin"
  host_catalog_id = boundary_host_catalog_plugin.azure_exmaple.id
  attributes_json = jsonencode({ "filter" = "tagName eq 'service-type' and tagValue eq 'database'" })
}

resource "boundary_host_set_plugin" "foodev" {
  name                  = "My foodev host set plugin"
  host_catalog_id       = boundary_host_catalog_plugin.azure_exmaple.id
  preferred_endpoints   = ["cidr:54.0.0.0/8"]
  sync_interval_seconds = 60
  attributes_json = jsonencode({
    "filter" = "tagName eq 'tag-key' and tagValue eq 'foo'",
    "filter" = "tagName eq 'application' and tagValue eq 'dev'",
  })
}