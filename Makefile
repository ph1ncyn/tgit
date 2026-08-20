BINARY := tgit
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build cross clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

cross:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out=dist/$(BINARY)-$$os-$$arch$$ext; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
	done

clean:
	rm -rf bin dist
