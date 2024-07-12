# Retrieve Roles
data "boundary_roles" "example" {
	scope_id = "id"
}

# Retrieve Roles with "test" in the name
data "boundary_roles" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
