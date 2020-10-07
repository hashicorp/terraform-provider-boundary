---
page_title: "boundary_target Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The target resource allows you to configure a Boundary target.
---

# Resource `boundary_target`

The target resource allows you to configure a Boundary target.

## Example Usage

```terraform
resource "boundary_scope" "global" {
  global_scope     = true
  scope_id         = "global"
  auto_create_role = true
}

resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.global.id
  auto_create_role = true
}

resource "boundary_scope" "project" {
  name             = "project_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.org.id
  auto_create_role = true
}

resource "boundary_host_catalog" "foo" {
  name        = "test"
  description = "test catalog"
  scope_id    = boundary_scope.project.id
  type        = "static"
}

resource "boundary_host" "foo" {
  name            = "foo"
  host_catalog_id = boundary_host_catalog.foo.id
  scope_id        = boundary_scope.project.id
  address         = "10.0.0.1"
}

resource "boundary_host" "bar" {
  name            = "bar"
  host_catalog_id = boundary_host_catalog.foo.id
  scope_id        = boundary_scope.project.id
  address         = "10.0.0.1"
}

resource "boundary_host_set" "foo" {
  name            = "foo"
  host_catalog_id = boundary_host_catalog.foo.id

  host_ids = [
    boundary_host.foo.id,
    boundary_host.bar.id,
  ]
}

resource "boundary_target" "foo" {
  name         = "foo"
  description  = "Foo target"
  default_port = "22"
  scope_id     = boundary_scope.project.id
  host_set_ids = [
    boundary_host_set.foo.id
  ]
}
```

## Schema

### Required

- **scope_id** (String, Required) The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
- **type** (String, Required) The target resource type.

### Optional

- **default_port** (Number, Optional) The default port for this target.
- **description** (String, Optional) The target description.
- **host_set_ids** (Set of String, Optional) A list of host set ID's.
- **id** (String, Optional) The ID of this resource.
- **name** (String, Optional) The target name. Defaults to the resource name.
- **session_connection_limit** (Number, Optional)
- **session_max_seconds** (Number, Optional)


