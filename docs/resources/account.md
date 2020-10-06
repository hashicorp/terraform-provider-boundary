---
page_title: "boundary_account Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The account resource allows you to configure a Boundary account.
---

# Resource `boundary_account`

The account resource allows you to configure a Boundary account.

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
```

## Schema

### Required

- **auth_method_id** (String, Required) The resource ID for the authentication method.
- **type** (String, Required) The resource type.

### Optional

- **description** (String, Optional) The account description.
- **id** (String, Optional) The ID of this resource.
- **login_name** (String, Optional) The login name for this account.
- **name** (String, Optional) The account name. Defaults to the resource name.
- **password** (String, Optional) The account password.


