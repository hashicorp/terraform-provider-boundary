# Retrieve Workers
data "boundary_workers" "example" {
	scope_id = "id"
}

# Retrieve Workers with "test" in the name
data "boundary_workers" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
