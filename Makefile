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
LINKFLAGS :=-s -X main.version=$(GIT_HASH) -extldflags "-static"
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
