---
page_title: "boundary_user Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The user resource allows you to configure a Boundary user.
---

# Resource `boundary_user`

The user resource allows you to configure a Boundary user.

## Example Usage

```terraform
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

## Schema

### Required

- **scope_id** (String, Required) The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.

### Optional

- **account_ids** (Set of String, Optional) Account ID's to associate with this user resource.
- **description** (String, Optional) The user description.
- **id** (String, Optional) The ID of this resource.
- **name** (String, Optional) The username. Defaults to the resource name.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_user.foo <my-id>
```
