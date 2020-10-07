---
page_title: "boundary_host_catalog Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The host catalog resource allows you to configure a Boundary host catalog. Host catalogs are always part of a project, so a project resource should be used inline or you should have the project ID in hand to successfully configure a host catalog.
---

# Resource `boundary_host_catalog`

The host catalog resource allows you to configure a Boundary host catalog. Host catalogs are always part of a project, so a project resource should be used inline or you should have the project ID in hand to successfully configure a host catalog.

## Example Usage

```terraform
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

resource "boundary_host_catalog" "example" {
  name        = "My catalog"
  description = "My first host catalog!"
  type        = "Static"
  scope_id    = boundary_scope.project.id
}
```

## Schema

### Required

- **scope_id** (String, Required) The scope ID in which the resource is created.
- **type** (String, Required) The host catalog type. Only `Static` (yes, title case) is supported.

### Optional

- **description** (String, Optional) The host catalog description.
- **id** (String, Optional) The ID of this resource.
- **name** (String, Optional) The host catalog name. Defaults to the resource name.


