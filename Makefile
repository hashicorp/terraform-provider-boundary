default: testacc
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
INSTALL_PATH=~/.local/share/terraform/plugins/localhost/providers/boundary/0.0.1/linux_$(GOARCH)
BUILD_ALL_PATH=${PWD}/bin

ifeq ($(GOOS), darwin)
	INSTALL_PATH=~/Library/Application\ Support/io.terraform/plugins/localhost/providers/boundary/0.0.1/darwin_$(GOARCH)
endif
ifeq ($(GOOS), "windows")
	INSTALL_PATH=%APPDATA%/HashiCorp/Terraform/plugins/localhost/providers/boundary/0.0.1/windows_$(GOARCH)
endif

REGISTRY_NAME?=docker.io/hashicorpboundary
IMAGE_NAME=postgres
IMAGE_TAG ?= $(REGISTRY_NAME)/$(IMAGE_NAME):11-alpine
DOCKER_ARGS ?= -d
PG_OPTS ?=
TEST_DB_PORT ?= 5432
BOUNDARY_VERSION = $(shell go mod edit -json | jq -r '.["Require"][] | select(.Path=="github.com/hashicorp/boundary") | .["Version"]')
GOPATH ?= $(abspath ~/go)
GOMODCACHE ?= $(GOPATH)/pkg/mod

tools:
	go generate -tags tools tools/tools.go
	go install github.com/hashicorp/copywrite@v0.15.0

test:
	echo "Placeholder"

# Run acceptance tests
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

dev:
	GOOS=${GOOS} GOARCH=${GOARCH} ./scripts/plugins.sh
	mkdir -p $(INSTALL_PATH)
	go build -o $(INSTALL_PATH)/terraform-provider-boundary main.go

dev-no-plugins:
	mkdir -p $(INSTALL_PATH)
	go build -o $(INSTALL_PATH)/terraform-provider-boundary main.go

all:
	mkdir -p $(BUILD_ALL_PATH)
	GOOS=darwin go build -o $(BUILD_ALL_PATH)/terraform-provider-boundary_darwin-amd64 main.go
	GOOS=windows go build -o $(BUILD_ALL_PATH)/terraform-provider-boundary_windows-amd64 main.go
	GOOS=linux go build -o $(BUILD_ALL_PATH)/terraform-provider-boundary_linux-amd64 main.go

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

rm-id-flag-from-docs:
	find docs/ -name "*.md" -type f | xargs sed -i -e '/- \*\*id\*\*/d'

test-database-up:
	@echo "Using image:                       $(IMAGE_TAG)"
	@echo "Additional postgres configuration: $(PG_OPTS)"
	@echo "Using volume:                      $(GOMODCACHE)/github.com/hashicorp/boundary@$(BOUNDARY_VERSION)/internal/db/schema/migrations:/migrations"
	@docker run \
		$(DOCKER_ARGS) \
		--name boundary-sql-tests \
		-p $(TEST_DB_PORT):5432 \
		-e POSTGRES_PASSWORD=boundary \
		-e POSTGRES_USER=boundary \
		-e POSTGRES_DB=boundary \
		-e PGDATA=/pgdata \
		--mount type=tmpfs,destination=/pgdata \
		-v "$(GOMODCACHE)/github.com/hashicorp/boundary@$(BOUNDARY_VERSION)/internal/db/schema/migrations":/migrations \
		$(IMAGE_TAG) \
		-c 'config_file=/etc/postgresql/postgresql.conf' \
		$(PG_OPTS) 1> /dev/null
	@echo "Test database available at:        127.0.0.1:$(TEST_DB_PORT)"
	@echo "For database logs run:"
	@echo "    docker logs boundary-sql-tests"

test-database-down:
	docker stop boundary-sql-tests || true
	docker rm -v boundary-sql-tests || true

.PHONY: testacc tools docs test-database-up test-database-down

.PHONY: copywrite
copywrite:
	copywrite headers

.PHONY: fmt
fmt:
	gofumpt -w $$(find . -name '*.go')

.PHONY: gen
gen: docs copywrite fmt
