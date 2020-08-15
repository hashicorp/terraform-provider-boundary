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
resource "boundary_user" "example" {
  name        = "My user"
  description = "My first user!"
}
```

Usage for non-default organization (users are organization level only resources):

```hcl
resource "boundary_user" "example" {
  name        = "My user"
  description = "My first user!"
  scope_id    = "o_1234567890"
}
```

## Argument Reference

The following arguments are optional:
* `name` - The username. Defaults to the resource name.
* `description` - The user description.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
