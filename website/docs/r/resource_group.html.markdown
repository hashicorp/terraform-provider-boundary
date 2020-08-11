---
layout: "boundary"
page_title: "Boundary: group_resource"
sidebar_current: "docs-boundary-group-resource"
description: |-
  Group resource for the Boundary Terraform provider.
---

# boundary_group_resource 
The group resource allows you to configure a Boundary group. 

## Example Usage

```hcl
resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
}
```

## Argument Reference

The following arguments are optional:
* `name` - The group name. Defaults to the resource name.
* `description` - The group description.

