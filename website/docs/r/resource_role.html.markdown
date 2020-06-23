---
layout: "watchtower"
page_title: "Watchtower: role_resource"
sidebar_current: "docs-watchtower-role-resource"
description: |-
  Role resource for the Watchtower Terraform provider.
---

# role_resource 
The role resource allows you to configure a Watchtower role. 

## Example Usage

```hcl
resource "watchtower_role" "example" {
  name        = "My role"
  description = "My first role!"
}
```

## Argument Reference

The following arguments are optional:
* `name` - The role name. Defaults to the resource name.
* `description` - The role description.

