export GOPATH ?= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOHOSTOS     ?= $(shell go env GOHOSTOS)
GOHOSTARCH   ?= $(shell go env GOHOSTARCH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint
GOLANGCI_LINT_VERSION := v1.63.4
VERSION ?= $(shell git describe --tags --abbrev=0 || git rev-parse --short HEAD)
GITSHA := $(shell git rev-parse HEAD)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
BUILDUSER := $(shell whoami)@$(shell hostname)
BUILDDATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifeq ($(GOHOSTARCH),amd64)
	ifeq ($(GOHOSTOS),$(filter $(GOHOSTOS),linux freebsd darwin windows))
		# Only supported on amd64
		test-flags := -race
	endif
endif

.PHONY: unused lint style test release

all: unused lint style test

build:
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) CGO_ENABLED=0 go build -ldflags="\
	-X github.com/prometheus/common/version.Version=$(VERSION) \
	-X github.com/prometheus/common/version.Revision=$(GITSHA) \
	-X github.com/prometheus/common/version.Branch=$(GITBRANCH) \
	-X github.com/prometheus/common/version.BuildUser=$(BUILDUSER) \
	-X github.com/prometheus/common/version.BuildDate=$(BUILDDATE)" \
	-o k8-ldap-configmap cmd/k8-ldap-configmap/main.go

test:
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go test $(test-flags) ./...

coverage:
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go test $(test-flags) -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

unused:
	@echo ">> running check for unused/missing packages in go.mod"
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go mod tidy
	@git diff --exit-code -- go.sum go.mod

lint: $(GOLANGCI_LINT)
	@echo ">> running golangci-lint"
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go list -e -compiled -test=true -export=false -deps=true -find=false -tags= -- ./... > /dev/null
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GOLANGCI_LINT) run ./...

style:
	@echo ">> checking code style"
	@fmtRes=$$(gofmt -d $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	if [ -n "$${fmtRes}" ]; then \
		echo "gofmt checking failed!"; echo "$${fmtRes}"; echo; \
		echo "Please ensure you are using $$($(GO) version) for formatting code."; \
		exit 1; \
	fi

format:
	go fmt ./...

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

release:
	@mkdir -p release
	@sed 's/:latest/:$(VERSION)/g' install/deployment.yaml > release/deployment.yaml
	@cp install/namespace-rbac.yaml release/namespace-rbac.yaml

bump-version:
	@grep -q '## $(VERSION)' CHANGELOG.md || { echo ">> Update CHANGELOG.md with version" ; exit 1; }
	@sed -i -e 's/version:.*/version: $(VERSION)/g' -e 's/appVersion:.*/appVersion: $(VERSION)/g' charts/k8-ldap-configmap/Chart.yaml
	@git add charts/k8-ldap-configmap/Chart.yaml
	@git add CHANGELOG.md
	@git commit -m "Release $(VERSION)"

tag:
	@git tag $(VERSION)
