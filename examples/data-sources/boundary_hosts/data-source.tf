# Retrieve Hosts
data "boundary_hosts" "example" {
	host_catalog_id = "id"
}

# Retrieve Hosts with "test" in the name
data "boundary_hosts" "example" {
	filter = "\"test\" in \"/item/name\""
	host_catalog_id = "id"
}
