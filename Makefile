CARTS := life snake
TINYGO_FLAGS := -target=wasi -scheduler=none -gc=leaking -opt=z -no-debug -panic=trap
ENGINE := github.com/stealthmonkey99/arkade

.PHONY: all test build fmt-check license-check manual-icons $(addprefix cart-,$(CARTS))

all: test

# Golden tests drive the checked-in .wasm, so no TinyGo is needed.
test:
	@set -e; for c in $(CARTS); do \
		echo "== $$c"; \
		(cd cartridges/$$c && go vet ./... && go test -count=1 ./...); \
	done
	cd tools/fleetlint && go test -count=1 ./...

build: $(addprefix cart-,$(CARTS))

# Rebuild cartridges/<slug>/<slug>.wasm: levels_gen.go from
# assets/*.arkadelevels, manifest_gen.go from manifest.json, TinyGo, then the
# wasmfix post-pass.
$(addprefix cart-,$(CARTS)): cart-%:
	cd cartridges/$* && go run $(ENGINE)/tools/genlevelpack .
	cd cartridges/$* && go run $(ENGINE)/sdk/tools/manifestgen .
	cd cartridges/$* && tinygo build $(TINYGO_FLAGS) -o $*.wasm .
	cd cartridges/$* && go run $(ENGINE)/tools/wasmfix -in $*.wasm -out $*.wasm

# gofmt over every tracked Go file. CI blocks on it.
fmt-check:
	@dirty=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$dirty" ]; then \
		echo "gofmt-dirty file(s):"; echo "$$dirty"; \
		echo "run: gofmt -w <file> and re-commit"; exit 1; \
	fi

# Every in-scope source file carries the SPDX licence line.
license-check:
	scripts/license-check.sh .

# Regenerate each cartridge's manual-icon.png from assets/icon.arkadesprite.png.
# The repo root has no go.mod, so it runs inside a cartridge module.
manual-icons:
	cd cartridges/$(firstword $(CARTS)) && ARKADE_CARTRIDGE_ROOTS=$(CURDIR)/cartridges go run $(ENGINE)/tools/manualicon
