---
layout: "boundary"
page_title: "Boundary: account_resource"
sidebar_current: "docs-boundary-account-resource"
description: |-
  Account resource for the Boundary Terraform provider.
---

# boundary_account_resource 
The account resource allows you to configure a Boundary account. 

## Example Usage

```hcl
resource "boundary_organization" "main" {}

resource "boundary_auth_method" "password" {
  scope_id = boundary_organization.main.id
  type     = "password"
}

resource "boundary_account" "jeff" {
  auth_method_id = boundary_auth_method.password.id
  type           = "password"
  login_name     = "jeff"
  password       = "$uper$ecure"
}  
```

## Argument Reference

The following arguments are required:
* `auth_method_id` - The resource ID for the authentication method.
* `type` - The resource type

The following arguments are optional:
* `description` - The account description.
* `name` - The account name. Defaults to the resource name.
* `login_name` - The login name for this account. 
* `password` - The account password. 

