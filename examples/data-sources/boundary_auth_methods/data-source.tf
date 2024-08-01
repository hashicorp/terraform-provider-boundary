# Retrieve AuthMethods
data "boundary_auth_methods" "example" {
	scope_id = "id"
}

# Retrieve AuthMethods with "test" in the name
data "boundary_auth_methods" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
