---
layout: "watchtower"
page_title: "Watchtower: host_resource"
sidebar_current: "docs-watchtower-host-resource"
description: |-
  Host resource for the Watchtower Terraform provider.
---

# host_resource 
The host resource allows you to configure a Watchtower host. Hosts are always 
part of a host catalog, so a host catalog resource should be used inline or you should have
the host catalog ID in hand to successfully configure a host. 

## Example Usage

```hcl
resource "watchtower_project" "example" {
  description = "My first project!"
}

resource "watchtower_host_catalog" "example" {
  name        = "My catalog"
  description = "My first host catalog!"
  type        = "Static"
  project_id  = watchtower_project.example.id
}

resource "watchtower_host" "example" {
  name             = "My host"
  description      = "My first host!"
  type             = "Static"
  host_catalog_id  = watchtower_host_catalog.example.id
}
```

## Argument Reference

The following arguments are required:
* `type` - The host type. Only `Static` (yes, title case) is supported.
* `host_catalog_id` - The host catalog in which to create this host.

The following arguments are optional:
* `name` - The host name. Defaults to the resource name.
* `description` - The host description.

