---
layout: "watchtower"
page_title: "Watchtower: group_resource"
sidebar_current: "docs-watchtower-group-resource"
description: |-
  Group resource for the Watchtower Terraform provider.
---

# watchtower_group_resource 
The group resource allows you to configure a Watchtower group. 

## Example Usage

```hcl
resource "watchtower_group" "example" {
  name        = "My group"
  description = "My first group!"
}
```

## Argument Reference

The following arguments are optional:
* `name` - The group name. Defaults to the resource name.
* `description` - The group description.

