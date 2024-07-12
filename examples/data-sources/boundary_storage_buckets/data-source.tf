# Retrieve StorageBuckets
data "boundary_storage_buckets" "example" {
	scope_id = "id"
}

# Retrieve StorageBuckets with "test" in the name
data "boundary_storage_buckets" "example" {
	filter = "\"test\" in \"/item/name\""
	scope_id = "id"
}
