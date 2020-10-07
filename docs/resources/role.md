---
page_title: "boundary_role Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The role resource allows you to configure a Boundary role.
---

# Resource `boundary_role`

The role resource allows you to configure a Boundary role.

## Example Usage

Basic usage:

```terraform
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  scope_id    = boundary_scope.org.id
}
```

Usage with a user resource:

```terraform
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_user" "foo" {
  name     = "User 1"
  scope_id = boundary_scope.org.id
}

resource "boundary_user" "bar" {
  name     = "User 2"
  scope_id = boundary_scope.org.id
}

resource "boundary_role" "example" {
  name        = "My role"
  description = "My first role!"
  principals  = [boundary_user.foo.id, boundary_user.bar.id]
  scope_id    = boundary_scope.org.id
}
```

Usage with user and grants resource:

```terraform
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
  scope_id    = boundary_scope.org.id
}
```

Usage for a project-specific role:

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

resource "boundary_user" "readonly" {
  name        = "readonly"
  description = "A readonly user"
  scope_id    = boundary_scope.org.id
}

resource "boundary_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  principals  = [boundary_user.readonly.id]
  grants      = ["id=*;action=read"]
  scope_id    = boundary_scope.project.id
}
```

## Schema

### Required

- **scope_id** (String, Required) The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.

### Optional

- **default_role** (Boolean, Optional) Indicates that the role containing this value is the default role (that is, has the id 'r_default'), which triggers some specialized behavior to allow it to be imported and managed.
- **description** (String, Optional) The role description.
- **grant_scope_id** (String, Optional)
- **grant_strings** (Set of String, Optional) A list of stringified grants for the role.
- **id** (String, Optional) The ID of this resource.
- **name** (String, Optional) The role name. Defaults to the resource name.
- **principal_ids** (Set of String, Optional) A list of principal (user or group) IDs to add as principals on the role.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_role.foo <my-id>
```