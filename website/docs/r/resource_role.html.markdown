---
layout: "boundary"
page_title: "Boundary: role_resource"
sidebar_current: "docs-boundary-role-resource"
description: |-
  Role resource for the Boundary Terraform provider.
---

# role_resource 
The role resource allows you to configure a Boundary role. 

## Example Usage
Basic usage:

```hcl
resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
}
```

Usage with a user resource:

```hcl
resource "boundary_user" "foo" {
  name = "User 1"
}

resource "boundary_user" "bar" {
  name = "User 2"
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  principals  = [boundary_user.foo.id, boundary_user.bar.id]
}

```

Usage with user and grants resource:

```hcl
resource "boundary_user" "readonly" {
  name = "readonly"
  description = "A readonly user"
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
}
```

## Argument Reference

The following arguments are optional:
* `name` - The role name. Defaults to the resource name.
* `description` - The role description.
* `principals` - A list of principal (user or group) IDs to add as principals on the role.
