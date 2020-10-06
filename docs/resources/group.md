---
page_title: "Boundary: group_resource"
subcategory: ""
description: |-
  Group resource for the Boundary Terraform provider.
---

# boundary_group_resource 
The group resource allows you to configure a Boundary group. 

## Example Usage

```hcl
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global" 
  auto_create_role = true
}

resource "boundary_user" "foo" {
  description = "foo user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
  member_ids  = [boundary_user.foo.id]
  scope_id    = boundary_scope.org.id
}
```

Usage for project-specific group:

```hcl
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global" 
  auto_create_role = true
}

resource "boundary_scope" "project" {
  name             = "project_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.org.id
  auto_create_role = true
}

resource "boundary_user" "foo" {
  description = "foo user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
  member_ids  = [boundary_user.foo.id]
  scope_id    = boundary_scope.project.id
}
```

## Argument Reference

The following arguments are optional:
* `description` - The group description.
* `name` - The group name. Defaults to the resource name.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
* `member_ids` - Resource IDs for group members, these are most likely boundary users.
