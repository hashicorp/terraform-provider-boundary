---
layout: "boundary"
page_title: "Boundary: organization_resource"
sidebar_current: "docs-boundary-organization-resource"
description: |-
  Organization resource for the Boundary Terraform provider.
---

# boundary_organization_resource 
The organization resource allows you to configure a Boundary organization. 

## Example Usage

```hcl
resource "boundary_organization" "foo" {
  name        = "foo_organization"
  description = "My first organization!"
}
```

Usage for organization-specific resources:

```hcl
resource "boundary_organization" "foo" {
  name = "foo_organization"
}

resource "boundary_user" "foo" {
  description = "foo user"
}

resource "boundary_group" "example" {
  name        = "My group"
  description = "My first group!"
  member_ids  = [boundary_user.foo.id]
  scope_id    = boundary_organization.foo.id
}
```

Usage with project resource and a group specific to that project:
```hcl
resource "boundary_organization" "foo" {
  name = "foo_organization"
}

resource "boundary_project" "foo" {
  description = "foo user"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_group" "foo" {
  scope_id = boundary_project.foo.id
}
```

## Argument Reference

The following arguments are optional:
* `description` - The organization description.
* `name` - The organization name. Defaults to the resource name.
