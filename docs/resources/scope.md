---
page_title: "boundary_scope Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The scope resource allows you to configure a Boundary scope.
---

# Resource `boundary_scope`

The scope resource allows you to configure a Boundary scope.

## Example Usage

Creating the global scope:

```terraform
resource "boundary_scope" "global" {
  global_scope     = true
  scope_id         = "global"
  auto_create_role = true
}
```

Creating an organization scope within global:

```terraform
resource "boundary_scope" "org" {
  name             = "organization_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.global.id
  auto_create_role = true
}
```

Creating an project scope within an organization:

```terraform
resource "boundary_scope" "project" {
  name             = "project_one"
  description      = "My first scope!"
  scope_id         = boundary_scope.org.id
  auto_create_role = true
}
```

Creating an organization scope with a managed role for administration (auto create role set false):

```terraform
resource "boundary_scope" "org" {
  name        = "organization_one"
  description = "My first scope!"
  scope_id    = boundary_scope.global.id
}

resource "boundary_role" "org_admin" {
  scope_id       = boundary_scope.global.id
  grant_scope_id = boundary_scope.org.id
  grant_strings  = ["id=*;type=*;actions=*"]
  principal_ids  = ["u_auth"]
}
```

## Schema

### Required

- **scope_id** (String, Required) The scope ID containing the sub scope resource.

### Optional

- **auto_create_admin_role** (Boolean, Optional) If set, when a new scope is created, the provider will not disable the functionality that automatically creates a role in the new scope and gives permissions to manage the scope to the provider's user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.
- **auto_create_default_role** (Boolean, Optional) If set, when a new scope is created, the provider will not disable the functionality that automatically creates a role in the new scope and gives listing of scopes and auth methods and the ability to authenticate to the anonymous user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.
- **description** (String, Optional) The scope description.
- **global_scope** (Boolean, Optional) Indicates that the scope containing this value is the global scope, which triggers some specialized behavior to allow it to be imported and managed.
- **name** (String, Optional) The scope name. Defaults to the resource name.

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_scope.foo <my-id>
```