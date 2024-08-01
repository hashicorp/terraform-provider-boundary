# Retrieve AuthTokens
data "boundary_auth_tokens" "example" {
	scope_id = "id"
}

# Retrieve AuthTokens with "test" in the name
data "boundary_auth_tokens" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
