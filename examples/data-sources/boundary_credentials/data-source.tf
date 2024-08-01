# Retrieve Credentials
data "boundary_credentials" "example" {
	credential_store_id = "id"
}

# Retrieve Credentials with "test" in the name
data "boundary_credentials" "example" {
	filter = "\"test\" in \"/item/name\""
	credential_store_id = "id"
}
