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

Usage for project-specific group:

```hcl
resource "boundary_project" "foo" {
  name = "foo_project"
}

resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
  scope_id    = boundary_project.foo.id
}
```

## Argument Reference

The following arguments are optional:
* `description` - The group description.
* `name` - The group name. Defaults to the resource name.
* `scope_id` - The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.
