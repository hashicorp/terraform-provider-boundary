default: testacc update-deps

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

update-deps:
	GOPROXY=direct GOSUMDB=off go get -u

dev:
	go build -o ~/.terraform.d/plugins/terraform-provider-watchtower main.go
