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
Basic usage:

```hcl
resource "watchtower_role" "example" {
  name        = "My role"
  description = "My first role!"
}
```

Usage with a user resource:

```hcl
resource "watchtower_user" "foo" {
  name = "User 1"
}

resource "watchtower_user" "bar" {
  name = "User 2"
}

resource "watchtower_role" "example" {
  name        = "My role"
  description = "My first role!"
  users       = [watchtower_user.foo.id, watchtower_user.bar.id]
}

```

Usage with user and grants resource:

```hcl
resource "watchtower_user" "readonly" {
  name = "readonly"
  description = "A readonly user"
}

resource "watchtower_role" "readonly" {
  name        = "readonly"
  description = "A readonly role"
  users       = [watchtower_user.readonly.id]
  grants      = ["id=*;action=read"]
}
```

## Argument Reference

The following arguments are optional:
* `name` - The role name. Defaults to the resource name.
* `description` - The role description.
* `users` - A list of user resource ID's to add as principles on the role.
* `groups` - A list of group resource ID's to add as principles on the role.
