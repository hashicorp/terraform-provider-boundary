---
page_title: "boundary_group Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The group resource allows you to configure a Boundary group.
---

# Resource `boundary_group`

The group resource allows you to configure a Boundary group.

## Example Usage

```terraform
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

```terraform
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

## Schema

### Required

- **scope_id** (String, Required) The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.

### Optional

- **description** (String, Optional) The group description.
- **id** (String, Optional) The ID of this resource.
- **member_ids** (Set of String, Optional) Resource IDs for group members, these are most likely boundary users.
- **name** (String, Optional) The group name. Defaults to the resource name.
