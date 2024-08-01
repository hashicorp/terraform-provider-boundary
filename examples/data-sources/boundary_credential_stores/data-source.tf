# Retrieve CredentialStores
data "boundary_credential_stores" "example" {
	scope_id = "id"
}

# Retrieve CredentialStores with "test" in the name
data "boundary_credential_stores" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
