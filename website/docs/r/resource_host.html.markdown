---
layout: "boundary"
page_title: "Boundary: host_resource"
sidebar_current: "docs-boundary-host-catalog-resource"
description: |-
  Host resource for the Boundary Terraform provider.
---

# host_resource 
The host resource allows you to configure a Boundary static host. Hosts are always part 
of a project, so a project resource should be used inline or you should have the project 
ID in hand to successfully configure a host. 

## Example Usage

```hcl
resource "boundary_organization" "foo" {}

resource "boundary_project" "example" {
  description = "My first project!"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_host" "example" {
  name        = "My catalog"
  description = "My first host!"
  address     = "10.0.0.1:8080"
  scope_id    = boundary_project.example.id
}
```

## Argument Reference

The following arguments are required:
* `scope_id` - The scope ID in which the resource is created.

The following arguments are optional:
* `name` - The host name. Defaults to the resource name.
* `description` - The host description.
* `address` - The static address of the host resource as <IP>:<port> or a domain name.
