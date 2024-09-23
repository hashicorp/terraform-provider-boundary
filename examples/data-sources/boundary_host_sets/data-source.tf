# Retrieve HostSets
data "boundary_host_sets" "example" {
	host_catalog_id = "id"
}

# Retrieve HostSets with "test" in the name
data "boundary_host_sets" "example" {
	filter = "\"test\" in \"/item/name\""
	host_catalog_id = "id"
}
