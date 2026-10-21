APP_NAME  := Monitor Controller
APP_ID    := com.github.sebastianrau.focusrite-mackie-control
# fyne package leaves the macOS category empty; it is set after packaging.
MACOS_CATEGORY := public.app-category.music
BIN       := monitor-control
CMD       := ./cmd/monitor-control
BUILD_DIR := build
DIST_DIR  := dist

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXE    := $(if $(filter windows,$(GOOS)),.exe,)

# VERSION is shown in the app and log; APP_VERSION is the plain x.y.z that
# platform packages require.
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
APP_VERSION ?= $(shell v=$$(git describe --tags --abbrev=0 2>/dev/null); echo $${v\#v} | grep . || echo 0.0.0)
LDFLAGS     := -s -w -X main.version=$(VERSION)

FYNE          ?= go run fyne.io/tools/cmd/fyne@v1.7.3
GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

ARTIFACT := $(DIST_DIR)/$(BIN)_$(VERSION)_$(GOOS)_$(GOARCH)

.DEFAULT_GOAL := help
.PHONY: help build run test cover lint fmt tidy package streamdeck-icons clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary for GOOS/GOARCH into build/
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BIN)$(EXE) $(CMD)

run: ## Run the app from source
	go run -ldflags "$(LDFLAGS)" $(CMD)

test: ## Run all tests with the race detector
	go test -race -count=1 ./...

cover: ## Run tests and open the coverage report
	go test -coverprofile=$(BUILD_DIR)/cover.out ./...
	go tool cover -html=$(BUILD_DIR)/cover.out

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

fmt: ## Format all Go files
	$(GOLANGCI_LINT) fmt ./...

tidy: ## Tidy go.mod/go.sum
	go mod tidy

package: build ## Build a distributable package for GOOS/GOARCH into dist/
	@mkdir -p $(DIST_DIR)
	$(FYNE) package --os $(GOOS) --src $(CMD) --exe $(CURDIR)/$(BUILD_DIR)/$(BIN)$(EXE) --name "$(APP_NAME)" \
		--app-id $(APP_ID) --app-version $(APP_VERSION) --icon $(CURDIR)/logo.png --release
ifeq ($(GOOS),darwin)
	plutil -replace LSApplicationCategoryType -string $(MACOS_CATEGORY) "$(APP_NAME).app/Contents/Info.plist"
	ditto -c -k --keepParent "$(APP_NAME).app" $(ARTIFACT).zip
	rm -rf "$(APP_NAME).app"
else ifeq ($(GOOS),linux)
	mv "$(APP_NAME).tar.xz" $(ARTIFACT).tar.xz
else
	powershell -NoProfile -Command "Compress-Archive -Force -Path '$(APP_NAME).exe' -DestinationPath '$(ARTIFACT).zip'"
	rm "$(APP_NAME).exe"
endif

streamdeck-icons: ## Rebuild the Stream Deck icon pack
	cd streamdeck && rm -f "Monitor Control Icons.streamDeckIconPack" && \
		zip -qr "Monitor Control Icons.streamDeckIconPack" com.github.sebastianraufocusrite-mackie-control.sdIconPack -x "*.DS_Store"

clean: ## Remove build output
	rm -rf $(BUILD_DIR) $(DIST_DIR)
