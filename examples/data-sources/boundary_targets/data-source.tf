# Retrieve Targets
data "boundary_targets" "example" {
	scope_id = "id"
}

# Retrieve Targets with "test" in the name
data "boundary_targets" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
