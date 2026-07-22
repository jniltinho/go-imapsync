BINARY  := dist/go-imapsync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
# Numeric version for archive names (strip leading v)
VERSION_NUM := $(shell echo '$(VERSION)' | sed 's/^v//')
LDFLAGS := -s -w -X go-imapsync/cmd.version=$(VERSION)

.PHONY: build test lint clean release release-cross docker-build live-dry

build:
	mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

test:
	go test -race ./...

lint:
	gofmt -l . | grep -v '^base/' | tee /dev/stderr | test -z "$$(cat)"
	go vet ./...
	@command -v staticcheck >/dev/null && staticcheck ./... || echo "staticcheck not installed; skipping (go install honnef.co/go/tools/cmd/staticcheck@latest)"

clean:
	rm -rf dist

# Host-native binary (optional UPX). Prefer release-cross for shipping.
release: build
	@if command -v upx >/dev/null; then \
		upx --best --lzma $(BINARY); \
	else \
		echo "warning: upx not installed; binary left uncompressed"; \
	fi
	@ls -lh $(BINARY)

# Cross-compile release binaries + archives (linux/amd64, darwin/arm64, windows/amd64).
# Artifacts land in dist/ as go-imapsync_<ver>_<os>_<arch>.tar.gz or .zip
release-cross:
	mkdir -p dist
	@echo "Building $(VERSION) (archives: $(VERSION_NUM))"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags '$(LDFLAGS)' \
		-o dist/go-imapsync_linux_amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags '$(LDFLAGS)' \
		-o dist/go-imapsync_darwin_arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags '$(LDFLAGS)' \
		-o dist/go-imapsync_windows_amd64.exe .
	@if command -v upx >/dev/null; then \
		upx --best --lzma dist/go-imapsync_linux_amd64 || true; \
	fi
	rm -rf dist/pkg && mkdir -p dist/pkg
	# linux/amd64
	cp dist/go-imapsync_linux_amd64 dist/pkg/go-imapsync
	tar -czvf dist/go-imapsync_$(VERSION_NUM)_linux_amd64.tar.gz \
		-C dist/pkg go-imapsync \
		-C $(CURDIR) LICENSE README.md CHANGELOG.md CONTRIBUTING.md examples/ docs/architecture/ docs/GOOD_PRACTICES.md
	# darwin/arm64 (Apple Silicon)
	cp dist/go-imapsync_darwin_arm64 dist/pkg/go-imapsync
	tar -czvf dist/go-imapsync_$(VERSION_NUM)_darwin_arm64.tar.gz \
		-C dist/pkg go-imapsync \
		-C $(CURDIR) LICENSE README.md CHANGELOG.md CONTRIBUTING.md examples/ docs/architecture/ docs/GOOD_PRACTICES.md
	# windows/amd64
	cp dist/go-imapsync_windows_amd64.exe dist/pkg/go-imapsync.exe
	cd dist/pkg && zip -q ../go-imapsync_$(VERSION_NUM)_windows_amd64.zip go-imapsync.exe \
		&& cd $(CURDIR) \
		&& zip -q dist/go-imapsync_$(VERSION_NUM)_windows_amd64.zip \
			LICENSE README.md CHANGELOG.md CONTRIBUTING.md \
			examples/* docs/GOOD_PRACTICES.md
	@# architecture HTML is large; include only on tar.gz unix packages above
	rm -rf dist/pkg
	@ls -lh dist/go-imapsync_$(VERSION_NUM)_*

docker-build:
	docker build -f deploy/docker/Dockerfile -t go-imapsync:$(VERSION) .

# Opt-in live dry-run (requires env vars; never runs in make test).
live-dry: build
	@test -n "$$GOIMAPSYNC_HOST1" || (echo "set GOIMAPSYNC_HOST1/USER1/PASSWORD1 and HOST2/USER2/PASSWORD2" >&2; exit 2)
	$(BINARY) \
		--host1 "$$GOIMAPSYNC_HOST1" --user1 "$$GOIMAPSYNC_USER1" --password1 "$$GOIMAPSYNC_PASSWORD1" \
		--host2 "$$GOIMAPSYNC_HOST2" --user2 "$$GOIMAPSYNC_USER2" --password2 "$$GOIMAPSYNC_PASSWORD2" \
		--dry --justfolders
