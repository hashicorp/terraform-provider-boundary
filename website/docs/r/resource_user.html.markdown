---
layout: "boundary"
page_title: "Boundary: user_resource"
sidebar_current: "docs-boundary-user-resource"
description: |-
  User resource for the Boundary Terraform provider.
---

# user_resource 
The user resource allows you to configure a Boundary user. 

## Example Usage

```hcl
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
```

## Argument Reference

The following arguments are optional:
* `account_ids` - Account ID's to associate with this user resource.
* `name` - The username. Defaults to the resource name.
* `description` - The user description.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
