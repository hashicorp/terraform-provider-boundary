---
page_title: "boundary_host_set Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The host_set resource allows you to configure a Boundary host set. Host sets are always part of a host catalog, so a host catalog resource should be used inline or you should have the host catalog ID in hand to successfully configure a host set.
---

# Resource `boundary_host_set`

The host_set resource allows you to configure a Boundary host set. Host sets are always part of a host catalog, so a host catalog resource should be used inline or you should have the host catalog ID in hand to successfully configure a host set.

## Example Usage

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

resource "boundary_host_catalog" "static" {
  scope_id = boundary_scope.project.id
}

resource "boundary_host" "1" {
  name            = "host_1"
  description     = "My first host!"
  address         = "10.0.0.1"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host" "2" {
  name            = "host_2"
  description     = "My second host!"
  address         = "10.0.0.2"
  host_catalog_id = boundary_host_catalog.static.id
  scope_id        = boundary_scope.project.id
}

resource "boundary_host_set" "web" {
  host_catalog_id = boundary_host_catalog.static.id
  host_ids = [
    boundary_host.1.id,
    boundary_host.2.id,
  ]
}
```

## Schema

### Required

- **host_catalog_id** (String, Required) The catalog for the hostset.
- **type** (String, Required)

### Optional

- **description** (String, Optional) The hostset description.
- **host_ids** (Set of String, Optional) The list of host IDs contained in this set.
- **name** (String, Optional) The hostset name. Defaults to the resource name.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_host_set.foo <my-id>
```
