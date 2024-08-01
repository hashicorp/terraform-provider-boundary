# Retrieve SessionRecordings with "test" in the name
data "boundary_session_recordings" "example" {
	filter = "\"test\" in \"/item/name\""
}
