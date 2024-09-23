# Retrieve CredentialLibraries
data "boundary_credential_libraries" "example" {
	credential_store_id = "id"
}

# Retrieve CredentialLibraries with "test" in the name
data "boundary_credential_libraries" "example" {
	filter = "\"test\" in \"/item/name\""
	credential_store_id = "id"
}
