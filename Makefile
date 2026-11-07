GO ?= $(shell command -v go 2>/dev/null || echo $(HOME)/Library/Go/1.27.3/bin/go)
VERSION ?= 3.0.0
BUILD_ID ?= $(shell date +%Y%m%d%H%M)

.PHONY: build test fmt vet helper finder package dev clean

build:
	$(GO) build ./...

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

helper:
	mkdir -p native/helper/bin
	xcrun swiftc -O -swift-version 5 -target arm64-apple-macos15.0 \
		-framework AppKit -o native/helper/bin/RClickHelper \
		native/helper/RClickHelper.swift

finder:
	mkdir -p native/finder/bin
	xcrun swiftc -O -parse-as-library -swift-version 5 \
		-target arm64-apple-macos15.0 -module-name RClickFinderSync \
		-framework FinderSync -framework Cocoa \
		-Xlinker -e -Xlinker _NSExtensionMain \
		-o native/finder/bin/RClickFinderSync \
		native/finder/FinderSyncExt.swift \
		native/finder/SharedConfigStore.swift \
		native/finder/models/RCBase.swift \
		native/finder/models/OpenWithApp.swift \
		native/finder/models/RCAction.swift \
		native/finder/models/NewFile.swift \
		native/finder/models/CommonDir.swift \
		native/finder/models/QuickCommand.swift \
		native/finder/shared/Constants.swift \
		native/finder/shared/AppLogger.swift \
		native/finder/shared/Utils.swift \
		native/finder/shared/Messager.swift

package: helper finder
	VERSION=$(VERSION) BUILD_ID=$(BUILD_ID) ./scripts/package-local.sh

install: helper finder
	INSTALL_DIR="$(HOME)/Applications" VERSION=$(VERSION) BUILD_ID=$(BUILD_ID) ./scripts/package-local.sh

dev: helper
	$(GO) tool mygo dev .

clean:
	rm -rf build native/helper/bin native/finder/bin