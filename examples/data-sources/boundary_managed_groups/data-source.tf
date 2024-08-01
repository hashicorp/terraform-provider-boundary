# Retrieve ManagedGroups
data "boundary_managed_groups" "example" {
	auth_method_id = "id"
}

# Retrieve ManagedGroups with "test" in the name
data "boundary_managed_groups" "example" {
	filter = "\"test\" in \"/item/name\""
	auth_method_id = "id"
}
