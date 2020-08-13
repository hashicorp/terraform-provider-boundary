---
layout: "boundary"
page_title: "Provider: Boundary"
sidebar_current: "docs-boundary-index"
description: |-
  Terraform provider Boundary.
---

# Boundary Provider

This provider configures Boundary. 

## Example Usage

Do not keep your authentication password in HCL for production environments, use Terraform environment variables.

```hcl
provider "boundary" {
  base_url             = "https://127.0.0.1:9200"
  default_scope        = "o_0000000000"
	auth_method_id       = "am_1234567890"
	auth_method_username = "foo"
	auth_method_password = "bar"
}
```
