# Retrieve Accounts
data "boundary_accounts" "example" {
	auth_method_id = "id"
}

# Retrieve Accounts with "test" in the name
data "boundary_accounts" "example" {
	filter = "\"test\" in \"/item/name\""
	auth_method_id = "id"
}
