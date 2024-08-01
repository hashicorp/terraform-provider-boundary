# Retrieve Sessions
data "boundary_sessions" "example" {
	scope_id = "id"
}

# Retrieve Sessions with "test" in the name
data "boundary_sessions" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
