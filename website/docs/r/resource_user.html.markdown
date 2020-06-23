---
layout: "watchtower"
page_title: "Watchtower: user_resource"
sidebar_current: "docs-watchtower-user-resource"
description: |-
  User resource for the Watchtower Terraform provider.
---

# user_resource 
The user resource allows you to configure a Watchtower user. 

## Example Usage

```hcl
resource "watchtower_user" "example" {
  name        = "My user"
  description = "My first user!"
}
```

## Argument Reference

The following arguments are optional:
* `name` - The username. Defaults to the resource name.
* `description` - The user description.

