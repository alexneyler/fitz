BINARY := fitz
PKG := ./cmd/fitz
DIST_DIR := dist
LOCAL_OS := $(shell go env GOOS)
LOCAL_ARCH := $(shell go env GOARCH)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

define asset_name
$(BINARY)_$(1)_$(2)$(if $(filter windows,$(1)),.exe,)
endef

.PHONY: build run release-local release clean fmt fmt-check vet lint test

build:
	@mkdir -p bin
	go build -o bin/$(BINARY) $(PKG)

run: build
	@./bin/$(BINARY) $(ARGS)

release-local:
	@mkdir -p $(DIST_DIR)
	GOOS=$(LOCAL_OS) GOARCH=$(LOCAL_ARCH) go build -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(call asset_name,$(LOCAL_OS),$(LOCAL_ARCH)) $(PKG)

release:
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		name="$(BINARY)_$${os}_$${arch}"; \
		if [ "$${os}" = "windows" ]; then name="$$name.exe"; fi; \
		echo "building $${os}/$${arch}"; \
		GOOS=$${os} GOARCH=$${arch} go build -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$$name $(PKG); \
	done

clean:
	rm -rf bin $(DIST_DIR)


fmt:
	@files=$$(git ls-files '*.go'); \
	if [ -n "$$files" ]; then gofmt -w $$files; fi

fmt-check:
	@files=$$(git ls-files '*.go'); \
	if [ -z "$$files" ]; then exit 0; fi; \
	unformatted=$$(gofmt -l $$files); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files need gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	go vet ./...

lint: fmt-check vet

test:
	go test ./...
