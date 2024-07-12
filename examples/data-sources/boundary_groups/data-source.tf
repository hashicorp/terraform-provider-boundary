# Retrieve Groups
data "boundary_groups" "example" {
	scope_id = "id"
}

# Retrieve Groups with "test" in the name
data "boundary_groups" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
