BINARY  := dist/go-imapsync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X go-imapsync/cmd.version=$(VERSION)

.PHONY: build test lint clean release docker-build live-dry

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

release: build
	@if command -v upx >/dev/null; then \
		upx --best --lzma $(BINARY); \
	else \
		echo "warning: upx not installed; binary left uncompressed"; \
	fi
	@ls -lh $(BINARY)

docker-build:
	docker build -f deploy/docker/Dockerfile -t go-imapsync:$(VERSION) .

# Opt-in live dry-run (requires env vars; never runs in make test).
live-dry: build
	@test -n "$$GOIMAPSYNC_HOST1" || (echo "set GOIMAPSYNC_HOST1/USER1/PASSWORD1 and HOST2/USER2/PASSWORD2" >&2; exit 2)
	$(BINARY) \
		--host1 "$$GOIMAPSYNC_HOST1" --user1 "$$GOIMAPSYNC_USER1" --password1 "$$GOIMAPSYNC_PASSWORD1" \
		--host2 "$$GOIMAPSYNC_HOST2" --user2 "$$GOIMAPSYNC_USER2" --password2 "$$GOIMAPSYNC_PASSWORD2" \
		--dry --justfolders
