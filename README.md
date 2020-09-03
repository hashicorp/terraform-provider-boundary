Terraform Provider Boundary 
==================

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
-	[Go](https://golang.org/doc/install) >= 1.12

Building The Provider
---------------------

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command: 
```sh
$ go install
```

Adding Dependencies
---------------------

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.


Using the provider
----------------------
Please see our detailed docs for individual resource usage. Below is a complex example using the Boundary provider to configure all resource types available:

```hcl
provider "boundary" {
  base_url             = "http://127.0.0.1:9200"
  auth_method_id       = "ampw_0000000000"
  auth_method_username = "foo"
  auth_method_password = "foofoofoo"
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

variable "frontend_server_ips" {
  type = set(string)
  default = [
    "10.0.0.1",
    "10.0.0.2",
  ]
}

variable "backend_server_ips" {
  type = set(string)
  default = [
    "10.1.0.1",
    "10.1.0.2",
  ]
}

resource "boundary_organization" "corp" {}

resource "boundary_user" "backend" {
  for_each    = var.backend_team
  name        = each.key
  description = "Backend user: ${each.key}"
  scope_id    = boundary_organization.corp.id
}

resource "boundary_user" "frontend" {
  for_each    = var.frontend_team
  name        = each.key
  description = "Frontend user: ${each.key}"
  scope_id    = boundary_organization.corp.id
}

resource "boundary_user" "leadership" {
  for_each    = var.leadership_team
  name        = each.key
  description = "WARNING: Managers should be read-only"
  scope_id    = boundary_organization.corp.id
}

// organiation level group for the leadership team
resource "boundary_group" "leadership" {
  name        = "leadership_team"
  description = "Organization group for leadership team"
  member_ids  = [for user in boundary_user.leadership : user.id]
  scope_id    = boundary_organization.corp.id
}

// add org-level role for readonly access
resource "boundary_role" "organization_readonly" {
  name        = "readonly"
  description = "Read-only role"
  principals  = [boundary_group.leadership.id]
  grants      = ["id=*;actions=read"]
  scope_id    = boundary_organization.corp.id
}

// add org-level role for administration access
resource "boundary_role" "organization_admin" {
  name        = "admin"
  description = "Administrator role"
  principals = concat(
    [for user in boundary_user.backend : user.id],
    [for user in boundary_user.frontend : user.id]
  )
  grants   = ["id=*;actions=create,read,update,delete"]
  scope_id = boundary_organization.corp.id
}

// create a project for core infrastructure
resource "boundary_project" "core_infra" {
  description = "Core infrastrcture"
  scope_id    = boundary_organization.corp.id
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

resource "boundary_host" "backend_servers_service" {
  for_each        = var.backend_server_ips
  name            = "backend_server_service_${each.value}"
  description     = "Backend server host for service port"
  address         = "${each.key}:9200"
  scope_id        = boundary_project.core_infra.id
  host_catalog_id = boundary_host_catalog.backend_servers.id
}

resource "boundary_host" "backend_servers_ssh" {
  for_each        = var.backend_server_ips
  name            = "backend_server_ssh_${each.value}"
  description     = "Backend server host for SSH port"
  address         = "${each.key}:22"
  scope_id        = boundary_project.core_infra.id
  host_catalog_id = boundary_host_catalog.backend_servers.id
}

resource "boundary_host" "frontend_servers_console" {
  for_each        = var.frontend_server_ips
  name            = "frontend_server_console_${each.value}"
  description     = "Frontend server host for console port"
  address         = "${each.key}:443"
  scope_id        = boundary_project.core_infra.id
  host_catalog_id = boundary_host_catalog.web_servers.id
}

resource "boundary_host" "frontend_servers_ssh" {
  for_each        = var.frontend_server_ips
  name            = "frontend_server_ssh_${each.value}"
  description     = "Frontend server host for SSH port"
  address         = "${each.key}:22"
  scope_id        = boundary_project.core_infra.id
  host_catalog_id = boundary_host_catalog.web_servers.id
}

resource "boundary_host_catalog" "web_servers" {
  name        = "web_servers"
  description = "Web servers for frontend team"
  type        = "Static"
  scope_id    = boundary_project.core_infra.id
}

resource "boundary_host_catalog" "backend_servers" {
  name        = "backend_servers"
  description = "Web servers for backend team"
  type        = "Static"
  scope_id    = boundary_project.core_infra.id
}

resource "boundary_host_set" "backend_servers_service" {
  name            = "backend_servers_service"
  description     = "Host set for services servers"
  host_catalog_id = boundary_host_catalog.backend_servers.id
  host_ids        = [for host in boundary_host.backend_servers_service : host.id]
}

resource "boundary_host_set" "backend_servers_ssh" {
  name            = "backend_servers_ssh"
  description     = "Host set for backend servers SSH access"
  host_catalog_id = boundary_host_catalog.backend_servers.id
  host_ids        = [for host in boundary_host.backend_servers_ssh : host.id]
}

resource "boundary_host_set" "frontend_servers_console" {
  name            = "frontend_servers_console"
  description     = "Host set for frontend servers console access"
  host_catalog_id = boundary_host_catalog.web_servers.id
  host_ids        = [for host in boundary_host.frontend_servers_console : host.id]
}

resource "boundary_host_set" "frontend_servers_ssh" {
  name            = "frontend_servers_ssh"
  description     = "Host set for frontend servers SSH access"
  host_catalog_id = boundary_host_catalog.web_servers.id
  host_ids        = [for host in boundary_host.frontend_servers_ssh : host.id]
}

resource "boundary_target" "frontend_servers_console" {
  name        = "frontend_servers_console"
  description = "Frontend console target"
  scope_id    = boundary_project.core_infra.id

  host_set_ids = [
    boundary_host_set.frontend_servers_console.id
  ]
}

resource "boundary_target" "frontend_servers_ssh" {
  name        = "frontend_servers_ssh"
  description = "Frontend SSH target"
  scope_id    = boundary_project.core_infra.id

  host_set_ids = [
    boundary_host_set.frontend_servers_ssh.id
  ]
}

resource "boundary_target" "backend_servers_service" {
  name        = "backend_servers_service"
  description = "Backend service target"
  scope_id    = boundary_project.core_infra.id

  host_set_ids = [
    boundary_host_set.backend_servers_service.id,
  ]
}

resource "boundary_target" "backend_servers_ssh" {
  name        = "backend_servers_ssh"
  description = "Backend SSH target"
  scope_id    = boundary_project.core_infra.id

  host_set_ids = [
    boundary_host_set.backend_servers_ssh.id,
  ]
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

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
