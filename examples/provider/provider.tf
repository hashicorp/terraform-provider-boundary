provider "boundary" {
  addr                            = "http://127.0.0.1:9200"
  auth_method_id                  = "ampw_1234567890" # changeme
  password_auth_method_login_name = "myuser"          # changeme
  password_auth_method_password   = "passpass"        # changeme
}

provider "boundary" {
  addr                            = "http://127.0.0.1:9200"
  password_auth_method_login_name = "myuser"
  password_auth_method_password   = "passpass"
}

provider "boundary" {
  addr                            = "http://127.0.0.1:9200"
  password_auth_method_login_name = "myuser"
  password_auth_method_password   = "passpass"
  scope_id                        = "s_1234567890"
}
