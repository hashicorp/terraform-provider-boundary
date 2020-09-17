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

resource "boundary_user" "example" {
  name        = "My user"
  description = "My first user!"
  scope_id    = boundary_scope.org.id 
}
```

## Argument Reference

The following arguments are optional:
* `name` - The username. Defaults to the resource name.
* `description` - The user description.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
