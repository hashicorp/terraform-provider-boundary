---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "boundary_credential_store_vault Resource - terraform-provider-boundary"
subcategory: ""
description: |-
  The credential store for Vault resource allows you to configure a Boundary credential store for Vault.
---

# boundary_credential_store_vault (Resource)

The credential store for Vault resource allows you to configure a Boundary credential store for Vault.

## Example Usage

```terraform
resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_scope" "project" {
  name                   = "project_one"
  description            = "My first scope!"
  scope_id               = boundary_scope.org.id
  auto_create_admin_role = true
}

resource "boundary_credential_store_vault" "example" {
  name        = "vault_store"
  description = "My first Vault credential store!"
  address     = "http://localhost:55001"
  token       = "s.0ufRo6XEGU2jOqnIr7OlFYP5"
  scope_id    = boundary_scope.project.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **address** (String) The address to Vault server
- **scope_id** (String) The scope for this credential store
- **token** (String) The Vault token

### Optional

- **ca_cert** (String) The Vault CA certificate to use
- **client_certificate** (String) The Vault client certificate
- **client_certificate_key** (String) The Vault client certificate key
- **description** (String) The Vault credential store description.
- **name** (String) The Vault credential store name. Defaults to the resource name.
- **namespace** (String) The namespace within Vault to use
- **tls_server_name** (String) The Vault TLS server name
- **tls_skip_verify** (Boolean) Whether or not to skip TLS verification

### Read-Only

- **client_certificate_key_hmac** (String) The Vault client certificate key hmac
- **id** (String) The ID of the Vault credential store.
- **token_hmac** (String) The Vault token hmac

## Import

Import is supported using the following syntax:

```shell
terraform import boundary_credential_store_vault.foo <my-id>
```