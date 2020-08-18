---
layout: "boundary"
page_title: "Boundary: project_resource"
sidebar_current: "docs-boundary-project-resource"
description: |-
  Project resource for the Boundary Terraform provider.
---

# boundary_project_resource 
The project resource allows you to configure a Boundary project. 

## Example Usage

```hcl
resource "boundary_project" "foo" {
  name        = "foo_project"
  description = "My first project!"
}
```

Usage for project-specific resources:

```hcl
resource "boundary_project" "foo" {
  name = "foo_project"
}

resource "boundary_user" "foo" {
  description = "foo user"
}

resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
  member_ids  = [boundary_user.foo.id]
  scope_id    = boundary_project.foo.id
}
```

## Argument Reference

The following arguments are optional:
* `description` - The project description.
* `name` - The project name. Defaults to the resource name.
