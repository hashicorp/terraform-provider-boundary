# Retrieve HostCatalogs
data "boundary_host_catalogs" "example" {
	scope_id = "id"
}

# Retrieve HostCatalogs with "test" in the name
data "boundary_host_catalogs" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
