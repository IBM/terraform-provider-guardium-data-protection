# Makefile for Terraform Provider Guardium Data Protection

# Detect OS and architecture
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
PLUGIN_DIR := ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/$(GOOS)_$(GOARCH)

default: build-all

# Build the provider for current platform
build:
	go build -trimpath -o terraform-provider-guardium-data-protection

# Build for specific platform
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -trimpath -o dist/terraform-provider-guardium-data-protection_linux_amd64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/terraform-provider-guardium-data-protection_darwin_arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -trimpath -o dist/terraform-provider-guardium-data-protection_darwin_amd64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -trimpath -o dist/terraform-provider-guardium-data-protection_windows_amd64.exe

# Build for all supported platforms
build-all: build-linux-amd64 build-darwin-arm64 build-darwin-amd64 build-windows-amd64
	@echo "Built for all supported platforms (linux_amd64, darwin_arm64, darwin_amd64, windows_amd64)"

# # Install the provider locally for testing (auto-detects platform)
# install: build
# 	mkdir -p $(PLUGIN_DIR)
# 	cp terraform-provider-guardium-data-protection $(PLUGIN_DIR)/

# # Install for specific platforms
# install-linux-amd64: build-linux-amd64
# 	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/linux_amd64
# 	cp dist/terraform-provider-guardium-data-protection_linux_amd64 ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/linux_amd64/terraform-provider-guardium-data-protection

# install-darwin-arm64: build-darwin-arm64
# 	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_arm64
# 	cp dist/terraform-provider-guardium-data-protection_darwin_arm64 ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_arm64/terraform-provider-guardium-data-protection

# install-darwin-amd64: build-darwin-amd64
# 	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_amd64
# 	cp dist/terraform-provider-guardium-data-protection_darwin_amd64 ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_amd64/terraform-provider-guardium-data-protection

# install-windows-amd64: build-windows-amd64
# 	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/windows_amd64
# 	cp dist/terraform-provider-guardium-data-protection_windows_amd64.exe ~/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/windows_amd64/terraform-provider-guardium-data-protection.exe

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate documentation
docs:
	go generate ./...

# Clean build artifacts
clean:
	rm -f terraform-provider-guardium-data-protection
	rm -f terraform-provider-guardium-data-protection_*
	rm -rf dist/

# Download dependencies
deps:
	go mod download
	go mod tidy

# # Setup local provider mirror (for automatic platform detection)
# setup-mirror: build-all
# 	@echo "Setting up local provider mirror for supported platforms..."
# 	@mkdir -p $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/linux_amd64
# 	@mkdir -p $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_arm64
# 	@mkdir -p $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_amd64
# 	@mkdir -p $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/windows_amd64
# 	@cp dist/terraform-provider-guardium-data-protection_linux_amd64 $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/linux_amd64/terraform-provider-guardium-data-protection_v1.0.0
# 	@cp dist/terraform-provider-guardium-data-protection_darwin_arm64 $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_arm64/terraform-provider-guardium-data-protection_v1.0.0
# 	@cp dist/terraform-provider-guardium-data-protection_darwin_amd64 $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/darwin_amd64/terraform-provider-guardium-data-protection_v1.0.0
# 	@cp dist/terraform-provider-guardium-data-protection_windows_amd64.exe $(HOME)/.terraform.d/plugins/registry.terraform.io/ibm/guardium-data-protection/1.0.0/windows_amd64/terraform-provider-guardium-data-protection_v1.0.0.exe
# 	@echo "Provider mirror setup complete at: $(HOME)/.terraform.d/plugins/"
# 	@echo "Supported platforms: linux_amd64, darwin_arm64, darwin_amd64, windows_amd64"
# 	@echo "Terraform will now automatically use the correct platform binary"

# Run acceptance tests
testacc:
	TF_ACC=1 go test -v ./... -timeout 120m

.PHONY: build build-linux-amd64 build-darwin-arm64 build-darwin-amd64 build-windows-amd64 build-all test fmt lint docs clean deps testacc
# .PHONY: install install-linux-amd64 install-darwin-arm64 install-darwin-amd64 install-windows-amd64 setup-mirror