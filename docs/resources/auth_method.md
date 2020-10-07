---
page_title: "boundary_auth_method Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The auth method resource allows you to configure a Boundary auth_method.
---

# Resource `boundary_auth_method`

The auth method resource allows you to configure a Boundary auth_method.

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
```

## Schema

### Required

- **scope_id** (String, Required) The scope ID.
- **type** (String, Required) The resource type.

### Optional

- **description** (String, Optional) The auth method description.
- **id** (String, Optional) The ID of this resource.
- **min_login_name_length** (Number, Optional) The minimum login name length.
- **min_password_length** (Number, Optional) The minimum password length.
- **name** (String, Optional) The auth method name. Defaults to the resource name.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_auth_method.foo <my-id>
```
