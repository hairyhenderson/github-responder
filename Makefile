.DEFAULT_GOAL = build
extension = $(patsubst windows,.exe,$(filter windows,$(1)))
GO := go
PKG_NAME := github-responder
PREFIX := .
GO111MODULE := on
DOCKER_BUILDKIT ?= 1

DOCKER_REPO ?= hairyhenderson/github-responder
DOCKER_TAG ?= latest

ifeq ("$(CI)","true")
LINT_PROCS ?= 1
else
LINT_PROCS ?= $(shell nproc)
endif

COMMIT ?= `git rev-parse --short HEAD 2>/dev/null`
VERSION ?= `git describe --abbrev=0 --tags $(git rev-list --tags --max-count=1) 2>/dev/null | sed 's/v\(.*\)/\1/'`
# BUILD_DATE ?= `date -u +"%Y-%m-%dT%H:%M:%SZ"`

COMMIT_FLAG := -X `go list ./version`.GitCommit=$(COMMIT)
VERSION_FLAG := -X `go list ./version`.Version=$(VERSION)
# BUILD_DATE_FLAG := -X `go list ./version`.BuildDate=$(BUILD_DATE)

GOOS ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\1/')
GOARCH ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\2/')

# platforms := linux-amd64 linux-arm linux-arm64 darwin-amd64 windows-amd64.exe
platforms := linux-amd64

clean:
	rm -Rf $(PREFIX)/bin/*
	rm -f $(PREFIX)/*.[ci]id

build-x: $(patsubst %,$(PREFIX)/bin/$(PKG_NAME)_%,$(platforms))

$(PREFIX)/bin/$(PKG_NAME)_%_checksum.txt: $(PREFIX)/bin/$(PKG_NAME)_%
	@sha256sum $< > $@

$(PREFIX)/bin/checksums.txt: $(patsubst %,$(PREFIX)/bin/$(PKG_NAME)_%_checksum.txt,$(platforms))
	@cat $^ > $@

$(PREFIX)/%.signed: $(PREFIX)/%
	@keybase sign < $< > $@

%.iid: Dockerfile
	@docker build \
		--build-arg VCS_REF=$(COMMIT) \
		--build-arg CODEOWNERS="$(shell grep `dirname $@` .github/CODEOWNERS | cut -f2)" \
		--build-arg VERSION=$(VERSION) \
		--target $(subst .iid,,$@) \
		--iidfile $@ \
		.

v%-alpine.tag: alpine.iid
	@docker tag $(shell cat $^) $(DOCKER_REPO):$(subst .tag,,$@)
	@echo $(DOCKER_REPO):$(subst .tag,,$@) > $@

v%.tag: latest.iid
	@docker tag $(shell cat $^) $(DOCKER_REPO):$(subst .tag,,$@)
	@echo $(DOCKER_REPO):$(subst .tag,,$@) > $@

%.tag: %.iid
	@docker tag $(shell cat $^) $(DOCKER_REPO):$(subst .tag,,$@)
	@echo $(DOCKER_REPO):$(subst .tag,,$@) > $@

%.cid: %.iid
	@docker create --cidfile $@ $(shell cat $<)

build-release: artifacts.cid
	@docker cp $(shell cat $<):/bin/. bin/

docker-images: $(PKG_NAME).iid

$(PREFIX)/bin/$(PKG_NAME)_%: $(shell find $(PREFIX) -type f -name '*.go')
	GOOS=$(shell echo $* | cut -f1 -d-) GOARCH=$(shell echo $* | cut -f2 -d- | cut -f1 -d.) CGO_ENABLED=0 \
		$(GO) build \
			-ldflags "-w -s $(COMMIT_FLAG) $(VERSION_FLAG)" \
			-o $@ \
			./cmd/$(PKG_NAME)

$(PREFIX)/bin/$(PKG_NAME)$(call extension,$(GOOS)): $(PREFIX)/bin/$(PKG_NAME)_$(GOOS)-$(GOARCH)$(call extension,$(GOOS))
	cp $< $@

build: $(PREFIX)/bin/$(PKG_NAME)$(call extension,$(GOOS))

test:
	$(GO) test -v -race -coverprofile=c.out ./...

integration: ./bin/$(PKG)
	$(GO) test -v -tags=integration \
		./tests/integration -check.v

integration.iid: Dockerfile.integration $(PREFIX)/bin/$(PKG_NAME)_linux-amd64$(call extension,$(GOOS))
	docker build -f $< --iidfile $@ .

test-integration-docker: integration.iid
	docker run -it --rm $(shell cat $<)

gen-changelog:
	docker run -it -v $(shell pwd):/app --workdir /app -e CHANGELOG_GITHUB_TOKEN hairyhenderson/github_changelog_generator \
		github_changelog_generator --no-filter-by-milestone --exclude-labels duplicate,question,invalid,wontfix,admin

lint:
	@golangci-lint run --verbose --max-same-issues=0 --max-issues-per-linter=0

ci-lint:
	@golangci-lint run --verbose --max-same-issues=0 --max-issues-per-linter=0

.PHONY: gen-changelog clean test build-x build-release build test-integration-docker gen-docs lint clean-images clean-containers docker-images update-deps
.DELETE_ON_ERROR:
.SECONDARY:
