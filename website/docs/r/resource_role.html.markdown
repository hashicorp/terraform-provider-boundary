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
resource "boundary_organization" "foo" {}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  scope_id    = boundary_organization.foo.id
}
```

Usage with a user resource:

```hcl
resource "boundary_organization" "foo" {}

resource "boundary_user" "foo" {
  name     = "User 1"
  scope_id = boundary_organization.foo.id
}

resource "boundary_user" "bar" {
  name     = "User 2"
  scope_id = boundary_organization.foo.id
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  principals  = [boundary_user.foo.id, boundary_user.bar.id]
  scope_id    = boundary_organization.foo.id
}

```

Usage with user and grants resource:

```hcl
resource "boundary_organization" "foo" {}

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
  scope_id    = boundary_organization.foo.id
}
```

Usage for a project-specific role:

```hcl
resource "boundary_organization" "foo" {}

resource "boundary_project" "foo" {
  name     = "foo_project"
  scope_id = boundary_organization.foo.id
}

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
  scope_id    = boundary_project.foo.id
}
```

## Argument Reference

The following arguments are optional:
* `description` - The role description.
* `grants` - A list of stringified grants for the role.
* `name` - The role name. Defaults to the resource name.
* `principals` - A list of principal (user or group) IDs to add as principals on the role.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
