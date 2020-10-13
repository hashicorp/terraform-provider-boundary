---
page_title: "boundary_host Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The host resource allows you to configure a Boundary static host. Hosts are always part of a project, so a project resource should be used inline or you should have the project ID in hand to successfully configure a host.
---

# Resource `boundary_host`

The host resource allows you to configure a Boundary static host. Hosts are always part of a project, so a project resource should be used inline or you should have the project ID in hand to successfully configure a host.

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

resource "boundary_host" "example" {
  name        = "My catalog"
  description = "My first host!"
  address     = "10.0.0.1"
  scope_id    = boundary_scope.project.id
}
```

## Schema

### Required

- **host_catalog_id** (String, Required)
- **type** (String, Required)

### Optional

- **address** (String, Optional) The static address of the host resource as `<IP>` (note: port assignment occurs in the target resource definition, do not add :port here) or a domain name.
- **description** (String, Optional) The host description.
- **id** (String, Optional) The ID of this resource.
- **name** (String, Optional) The host name. Defaults to the resource name.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_host.foo <my-id>
```
