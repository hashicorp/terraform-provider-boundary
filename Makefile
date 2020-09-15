default: update-deps testacc 

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

testacc-ci: install-go
	git config --global --add url."git@github.com:".insteadOf "https://github.com/"
	TF_ACC=1 ~/.go/bin/go test ./... -v $(TESTARGS) -timeout 120m

install-go:
	./ci/goinstall.sh
	
update-deps:
	GOPROXY=direct GOSUMDB=off go get -u

dev:
	go build -o ~/.terraform.d/plugins/terraform-provider-boundary main.go
