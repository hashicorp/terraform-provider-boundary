default: update-deps testacc 
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
INSTALL_PATH=~/.local/share/terraform/plugins/localhost/providers/boundary/0.0.1/linux_$(GOARCH)
ifeq ($(GOOS), "darwin")
	INSTALL_PATH=~/Library/Application Support/io.terraform/plugins/localhost/providers/boundary/0.0.1/darwin_$(GOARCH)
endif
ifeq ($(GOOS), "windows")
	INSTALL_PATH=%APPDATA%/HashiCorp/Terraform/plugins/localhost/providers/boundary/0.0.1/windows_$(GOARCH)
endif

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
	mkdir -p $(INSTALL_PATH)	
	go build -o $(INSTALL_PATH)/terraform-provider-boundary main.go
