---
page_title: "Boundary: scope_resource"
subcategory: ""
description: |-
  Scope resource for the Boundary Terraform provider.
---

# boundary_scope_resource 
The scope resource allows you to configure a Boundary scope. 

## Example Usage

Creating the global scope:

```hcl
resource "boundary_scope" "global" {
  global_scope     = true
  scope_id         = "global"
  auto_create_role = true
}
```

Creating an organization scope within global:

```hcl
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.global.id
  auto_create_role = true
}
```

Creating an project scope within an organization:

```hcl
resource "boundary_scope" "project" {
  name             = "project_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.org.id
  auto_create_role = true
}
```

Creating an organization scope with a managed role for administration (auto create role set false):

```hcl
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.global.id
}

resource "boundary_role" "org_admin" {
  scope_id       = boundary_scope.global.id
  grant_scope_id = boundary_scope.org.id
  grant_strings  = ["id=*;type=*;actions=*"]
  principal_ids  = ["u_auth"]
}
```

## Argument Reference

The following arguments are optional:
* `description` - The scope description.
* `name` - The scope name. Defaults to the resource name.
* `global_scope` - Indicates that the scope containing this value is the global scope, which triggers some specialized behavior to allow it to be imported and managed. 
* `auto_create_role` - If set, when a new scope is created, the provider will not disable the functionality that automatically crates a role in the new scope and gives permissions to manage the scope to the provider's user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.
* `scope_id` - The scope ID containing the sub scope resource.
