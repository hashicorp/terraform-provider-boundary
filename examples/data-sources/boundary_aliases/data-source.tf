# Retrieve Aliases
data "boundary_aliases" "example" {
	scope_id = "id"
}

# Retrieve Aliases with "test" in the name
data "boundary_aliases" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
