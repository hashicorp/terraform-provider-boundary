# Retrieve Users
data "boundary_users" "example" {
	scope_id = "id"
}

# Retrieve Users with "test" in the name
data "boundary_users" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
