---
layout: "watchtower"
page_title: "Provider: Watchtower"
sidebar_current: "docs-watchtower-index"
description: |-
  Terraform provider Watchtower.
---

# Watchtower Provider

This provider configures Watchtower. 

## Example Usage

Do not keep your authentication password in HCL for production environments, use Terraform environment variables.

```hcl
provider "watchtower" {
  base_url             = "https://127.0.0.1:9200"
  default_organization = "o_0000000000"
	auth_method_id       = "am_1234567890"
	auth_method_username = "foo"
	auth_method_password = "bar"
}
```
