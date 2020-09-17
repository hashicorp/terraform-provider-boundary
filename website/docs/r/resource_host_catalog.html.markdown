---
layout: "boundary"
page_title: "Boundary: host_catalog_resource"
sidebar_current: "docs-boundary-host-catalog-resource"
description: |-
  Host catalog resource for the Boundary Terraform provider.
---

# host_catalog_resource 
The host catalog resource allows you to configure a Boundary host catalog. Host catalogs
are always part of a project, so a project resource should be used inline or you should have
the project ID in hand to successfully configure a host catalog. 

## Example Usage

```hcl
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

## Argument Reference

The following arguments are required:
* `type` - The host catalog type. Only `Static` (yes, title case) is supported.
* `scope_id` - The scope ID in which the resource is created.

The following arguments are optional:
* `name` - The host catalog name. Defaults to the resource name.
* `description` - The host catalog description.
