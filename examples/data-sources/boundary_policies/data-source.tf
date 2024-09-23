# Retrieve Policies
data "boundary_policies" "example" {
	scope_id = "id"
}

# Retrieve Policies with "test" in the name
data "boundary_policies" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
