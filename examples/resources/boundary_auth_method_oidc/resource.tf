resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_auth_method_oidc" "vault" {
  api_url_prefix     = "https://XO-XO-XO-XO-XOXOXO.boundary.hashicorp.cloud:9200"
  client_id          = "eieio"
  client_secret      = "hvo_secret_XO"
  description        = "My Boundary OIDC Auth Method for Vault"
  issuer             = "https://XO-XO-XO-XO-XOXOXO.vault.hashicorp.cloud:8200/v1/identity/oidc/provider/my-provider"
  scope_id           = "global"
  signing_algorithms = ["RS256"]
  type               = "oidc"
}
