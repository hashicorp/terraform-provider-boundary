---
layout: "boundary"
page_title: "Boundary: auth_method_resource"
sidebar_current: "docs-boundary-auth-method-resource"
description: |-
  Auth Method resource for the Boundary Terraform provider.
---

# boundary_auth_method_resource 
The auth method resource allows you to configure a Boundary auth_method. 

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
```

## Argument Reference

The following arguments are required:
* `scope_id` - The scope ID. 
* `type` - The resource type.

The following arguments are optional:
* `description` - The auth method description.
* `name` - The auth method name. Defaults to the resource name.
* `min_login_name_length` - The minimum login name length.
* `min_password_length` - The minimum password length.

