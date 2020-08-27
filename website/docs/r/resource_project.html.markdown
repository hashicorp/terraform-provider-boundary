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
resource "boundary_organization" "foo" {}

resource "boundary_project" "foo" {
  name        = "foo_project"
  description = "My first project!"
  scope_id    = boundary_organization.foo.id
}
```

Usage for project-specific resources:

```hcl
resource "boundary_organization" "foo" {}

resource "boundary_project" "foo" {
  name     = "foo_project"
  scope_id = boundary_organization.foo.id
}

resource "boundary_user" "foo" {
  description = "foo user"
  scope_id    = boundary_organization.foo.id
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
* `scope_id` - The scope for the project which is always an organization ID.
