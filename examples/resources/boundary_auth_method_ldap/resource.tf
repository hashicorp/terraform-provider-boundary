resource "boundary_scope" "org" {
  name                     = "organization_one"
  description              = "My first scope!"
  scope_id                 = "global"
  auto_create_admin_role   = true
  auto_create_default_role = true
}

resource "boundary_auth_method_ldap" "forumsys_ldap" {
  name          = "forumsys public LDAP"
  scope_id      = "global"                               # add the new auth method to the global scope
  urls          = ["ldap://ldap.forumsys.com"]           # the addr of the LDAP server
  user_dn       = "dc=example,dc=com"                    # the basedn for users
  user_attr     = "uid"                                  # the user attribute
  group_dn      = "dc=example,dc=com"                    # the basedn for groups
  bind_dn       = "cn=read-only-admin,dc=example,dc=com" # the dn to use when binding
  bind_password = "password"                             # passwd to use when binding
  state         = "active-public"                        # make sure the new auth-method is available to everyone
  enable_groups = true                                   # this turns-on the discovery of a user's groups
  discover_dn   = true                                   # this turns-on the discovery of an authenticating user's dn
}
