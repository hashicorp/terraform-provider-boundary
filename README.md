![](boundary.png)

Terraform Provider Boundary
==================

Available in the [Terraform Registry](https://registry.terraform.io/providers/hashicorp/boundary/latest).

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
-	[Go](https://golang.org/doc/install) >= 1.16

Building The Provider
---------------------

1. Clone the repository
1. Enter the repository directory
1. Build the provider using `make dev`. This will place the provider onto your system in a [Terraform 0.13-compliant](https://www.terraform.io/upgrade-guides/0-13.html#in-house-providers) manner.

You'll need to ensure that your Terraform file contains the information necessary to find the plugin when running `terraform init`. `make dev` will use a version number of 0.0.1, so the following block will work:

```hcl
terraform {
        required_providers {
                boundary = {
                        source = "localhost/providers/boundary"
                        version = "0.0.1"
                }
        }
}
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

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

In order to run the full suite of Acceptance tests,
a postgres docker container must be started first:

```sh
$ go mod download # ensure boundary is installed, files are used by the docker image
$ make test-database-up
```

Once the test database is ready the tests can be run using `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

For more details on the docker image and troubleshooting see the
[boundary testing doc](https://github.com/hashicorp/boundary/blob/main/CONTRIBUTING.md#testing).

Generating Docs
----------------------

From the root of the repo run:

```
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
```

Using the provider
----------------------
Please see our detailed docs for individual resource usage. Below is a complex example using the Boundary provider to configure all resource types available:

```hcl
provider "boundary" {
  addr                            = "http://127.0.0.1:9200"
  auth_method_id                  = "ampw_1234567890"      # changeme
  password_auth_method_login_name = "myuser"               # changeme
  password_auth_method_password   = "passpass"             # changeme
}

variable "users" {
  type    = set(string)
  default = [
    "Jim",
    "Mike",
    "Todd",
    "Jeff",
    "Randy",
    "Susmitha"
  ]
}

variable "readonly_users" {
  type    = set(string)
  default = [
    "Jeff",
    "Pete",
    "JT"
  ]
}

variable "backend_server_ips" {
  type    = set(string)
  default = [
    "10.1.0.1",
    "10.1.0.2",
  ]
}

resource "boundary_scope" "global" {
  global_scope = true
  scope_id     = "global"
}

resource "boundary_scope" "corp" {
  scope_id                 = boundary_scope.global.id
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_user" "users" {
  for_each    = var.users
  name        = each.key
  description = "User resource for ${each.key}"
  scope_id    = boundary_scope.corp.id
}

// organization level group for readonly users 
resource "boundary_group" "readonly" {
  name        = "readonly"
  description = "Organization group for readonly users"
  member_ids  = [for user in boundary_user.readonly_users : user.id]
  scope_id    = boundary_scope.corp.id
}

// add org-level role for readonly access
resource "boundary_role" "organization_readonly" {
  name        = "readonly"
  description = "Read-only role"
  principal_ids = [boundary_group.readonly_users.id]
  grant_strings = ["id=*;type=*;actions=read"]
  scope_id    = boundary_scope.corp.id
}

// add org-level role for administration access
resource "boundary_role" "organization_admin" {
  name        = "admin"
  description = "Administrator role"
  principal_ids = concat(
    [for user in boundary_user.user: user.id]
  )
  grant_strings   = ["id=*;type=*;actions=create,read,update,delete"]
  scope_id = boundary_scope.corp.id
}

// create a project for core infrastructure
resource "boundary_scope" "core_infra" {
  description              = "Core infrastrcture"
  scope_id                 = boundary_scope.corp.id
  auto_create_admin_role   = true
}

resource "boundary_host_catalog" "backend_servers" {
  name        = "backend_servers"
  description = "Backend servers host catalog"
  type        = "static"
  scope_id    = boundary_scope.core_infra.id
}

resource "boundary_host" "backend_servers" {
  for_each        = var.backend_server_ips
  type            = "static"
  name            = "backend_server_service_${each.value}"
  description     = "Backend server host"
  address         = "${each.key}"
  host_catalog_id = boundary_host_catalog.backend_servers.id
}

resource "boundary_host_set" "backend_servers_ssh" {
  type            = "static"
  name            = "backend_servers_ssh"
  description     = "Host set for backend servers"
  host_catalog_id = boundary_host_catalog.backend_servers.id
  host_ids        = [for host in boundary_host.backend_servers : host.id]
}

// create target for accessing backend servers on port :8000
resource "boundary_target" "backend_servers_service" {
  type         = "tcp"
  name         = "backend_servers_service"
  description  = "Backend service target"
  scope_id     = boundary_scope.core_infra.id
  default_port = "8080"

  host_set_ids = [
    boundary_host_set.backend_servers.id
  ]
}

// create target for accessing backend servers on port :22
resource "boundary_target" "backend_servers_ssh" {
  type         = "tcp"
  name         = "backend_servers_ssh"
  description  = "Backend SSH target"
  scope_id     = boundary_scope.core_infra.id
  default_port = "22"

  host_set_ids = [
    boundary_host_set.backend_servers_ssh.id
  ]
}
```
