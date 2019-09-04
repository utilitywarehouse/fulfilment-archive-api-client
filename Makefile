# get name of directory containing this Makefile
# (stolen from https://stackoverflow.com/a/18137056)
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
base_dir := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))

SERVICE ?= $(base_dir)
BUILDENV :=
BUILDENV += GO111MODULE=on
BUILDENV += CGO_ENABLED=0 
GIT_HASH := $(CIRCLE_SHA1)
ifeq ($(GIT_HASH),)
  GIT_HASH := $(shell git rev-parse HEAD)
endif
LINKFLAGS :=-s -X main.gitHash=$(GIT_HASH) -extldflags "-static"
TESTFLAGS := -v -cover
LINT_EXCLUDE=pb.go|pb.gw.go
LINT_FLAGS :=--disable-all --enable=vet --enable=vetshadow --enable=golint --enable=ineffassign --enable=deadcode  --enable=gosimple --enable=goconst --enable=gofmt --deadline=120s
LINTER_EXE := golangci-lint
LINTER := $(GOPATH)/bin/$(LINTER_EXE)

EMPTY :=
SPACE := $(EMPTY) $(EMPTY)
join-with = $(subst $(SPACE),$1,$(strip $2))

LEXC :=
ifdef LINT_EXCLUDE
	LEXC := $(call join-with,|,$(LINT_EXCLUDE))
endif

.DEFAULT_GOAL := all



.PHONY: install_packages
install_packages:
	go get -t -v ./... 2>&1 | sed -e "s/[[:alnum:]]*:x-oauth-basic/redacted/"

.PHONY: update_packages
update_packages:
	go get -u -t -v ./... 2>&1 | sed -e "s/[[:alnum:]]*:x-oauth-basic/redacted/"


$(LINTER):
	GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: $(LINTER)
ifdef LEXC
	$(LINTER) --exclude '$(LEXC)' run $(LINT_FLAGS) ./...
else
	$(LINTER) run $(LINT_FLAGS) ./...
endif

.PHONY: clean
clean:
	rm -f $(SERVICE)

# builds our binary
$(SERVICE): clean
	$(BUILDENV) go build -o $(SERVICE) -a -ldflags '$(LINKFLAGS)' ./cmd/$(SERVICE)

build: $(SERVICE)

.PHONY: test
test:
	$(BUILDENV) go test $(TESTFLAGS) ./...


.PHONY: all
all: clean install_packages $(LINTER) lint test build

UW_GITHUB := github.com/utilitywarehouse
BFAA_SCHEMA_DIR := $(GOPATH)/src/github.com/utilitywarehouse/finance-fulfilment-archive-api/proto
FULFILMENT_SCHEMA_DIR := $(GOPATH)/src/github.com/utilitywarehouse/finance-invoice-protobuf-model
ENVELOPE_SCHEMA_DIR=$(GOPATH)/src/github.com/utilitywarehouse/event-envelope-proto
BFAA_GEN_DIR := ./internal/pb/bfaa
FULFILMENT_GENERATED_DIR := ./internal/pb/fulfilment
PROTO_MAPPINGS := Mgoogle/protobuf/empty.proto=github.com/golang/protobuf/ptypes/empty,Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,Mgithub.com/utilitywarehouse/finance-invoice-protobuf-model/fulfilment.proto=github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/pb/fulfilment

protos:
	mkdir -pv $(BFAA_GEN_DIR)
	mkdir -pv $(FULFILMENT_GENERATED_DIR)

	GO111MODULE=off go get github.com/gogo/protobuf/protoc-gen-gogoslick
	GO111MODULE=off go get google.golang.org/grpc
	GO111MODULE=off go get -u $(UW_GITHUB)/finance-invoice-protobuf-model 2>&1 | sed -e "s/[[:alnum:]]*:x-oauth-basic/redacted/"
	GO111MODULE=off go get -u $(UW_GITHUB)/finance-fulfilment-archive-api 2>&1 | sed -e "s/[[:alnum:]]*:x-oauth-basic/redacted/"

	protoc \
	    --proto_path=$(FULFILMENT_SCHEMA_DIR) \
		--proto_path=$(ENVELOPE_SCHEMA_DIR) \
		--gogoslick_out=${PROTO_MAPPINGS}:${FULFILMENT_GENERATED_DIR} \
		$(FULFILMENT_SCHEMA_DIR)/fulfilment.proto \
		$(FULFILMENT_SCHEMA_DIR)/invoice.proto

	protoc \
		-I ${BFAA_SCHEMA_DIR} \
		-I ${FULFILMENT_SCHEMA_DIR} \
		-I ${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I .:${GOPATH}/src:${GOPATH}/src/github.com/gogo/protobuf/protobuf \
		--gogoslick_out=plugins=grpc,${PROTO_MAPPINGS}:${BFAA_GEN_DIR} \
		${BFAA_SCHEMA_DIR}/bill_fulfilment_archive_api.proto
