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
  auth_method_id       = "paum_1234567890"
  auth_method_username = "myuser"
  auth_method_password = "$uper$ecure9ass^^ord"
}
```

## Complex Usage

```hcl
provider "boundary" {
  base_url             = "https://127.0.0.1:9200"
  default_scope        = "o_0000000000"
  auth_method_id       = "paum_1234567890"
  auth_method_username = "myuser"
  auth_method_password = "$uper$ecure9ass^^ord"
}

variable "backend_team" {
  type = set(string)
  default = [
    "Jim Lambert",
    "Mike Gaffney",
    "Todd Knight",
  ]
}

variable "frontend_team" {
  type = set(string)
  default = [
    "Randy Morey",
    "Susmitha Girumala",
  ]
}

variable "leadership_team" {
  type = set(string)
  default = [
    "Jeff Mitchell",
    "Pete Pacent",
    "Jonathan Thomas (JT)",
    "Jeff Malnick"
  ]
}

resource "boundary_user" "backend" {
  for_each    = var.backend_team
  name        = each.key
  description = "Backend user: ${each.key}"
}

resource "boundary_user" "frontend" {
  for_each    = var.frontend_team
  name        = each.key
  description = "Frontend user: ${each.key}"
}

resource "boundary_user" "leadership" {
  for_each    = var.leadership_team
  name        = each.key
  description = "WARNING: Managers should be read-only"
}

// organiation level group for the leadership team
resource "boundary_group" "leadership" {
  name        = "leadership_team"
  description = "Organization group for leadership team"
  member_ids  = [for user in boundary_user.leadership : user.id]
}

// add org-level role for readonly access
resource "boundary_role" "organization_readonly" {
  name        = "readonly"
  description = "Read-only role"
  principals  = [boundary_group.leadership.id]
  grants      = ["id=*;actions=read"]
}

// add org-level role for administration access
resource "boundary_role" "organization_admin" {
  name        = "admin"
  description = "Administrator role"
  principals = concat(
    [for user in boundary_user.backend : user.id],
    [for user in boundary_user.frontend : user.id]
  )
  grants = ["id=*;actions=create,read,update,delete"]
}

// create a project for core infrastructure
resource "boundary_project" "core_infra" {
  description = "Core infrastrcture"
}

resource "boundary_group" "backend_core_infra" {
  name        = "backend"
  description = "Backend team group"
  member_ids  = [for user in boundary_user.backend : user.id]
  scope_id    = boundary_project.core_infra.id
}

resource "boundary_group" "frontend_core_infra" {
  name        = "frontend"
  description = "Frontend team group"
  member_ids  = [for user in boundary_user.frontend : user.id]
  scope_id    = boundary_project.core_infra.id
}

resource "boundary_host_catalog" "web_servers" {
  name        = "Web servers"
  description = "Web servers for frontend team"
  type        = "Static"
  scope_id    = boundary_project.core_infra.id
}

resource "boundary_host_catalog" "backend_servers" {
  name        = "Backend servers"
  description = "Web servers for backend team"
  type        = "Static"
  scope_id    = boundary_project.core_infra.id
}

// only allow the backend team access to the backend web servers host catalog
resource "boundary_role" "admin_backend_core_infra" {
  description = "Administrator role for backend core infrastructure"
  principals  = [boundary_group.backend_core_infra.id]
  grants      = ["id=${boundary_host_catalog.backend_servers.id};actions=create,read,update,delete"]
  scope_id    = boundary_project.core_infra.id
}

// only allow the frontend team access to the frontend web servers host catalog
resource "boundary_role" "admin_frontend_core_infra" {
  description = "Administrator role for frontend core infrastructure"
  principals  = [boundary_group.frontend_core_infra.id]
  grants      = ["id=${boundary_host_catalog.web_servers.id};actions=create,read,update,delete"]
  scope_id    = boundary_project.core_infra.id
}
```
