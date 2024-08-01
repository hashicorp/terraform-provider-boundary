# Retrieve Scopes
data "boundary_scopes" "example" {
	scope_id = "id"
}

# Retrieve Scopes with "test" in the name
data "boundary_scopes" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
