BINARY_NAME=spec-recon
VERSION=1.0.0
BUILD_DIR=dist

.PHONY: all build clean test test-e2e release

all: test build

build:
	go build -o $(BINARY_NAME) ./cmd/spec-recon

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf $(BUILD_DIR)
	rm -rf output/
	rm -f config_test.yaml
	rm -f spec-recon-test*

test:
	go test -v ./internal/...

test-e2e:
	go test -v ./test/e2e_test.go

release: clean
	mkdir -p $(BUILD_DIR)
	# Windows (AMD64)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/spec-recon
	# Linux (AMD64)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/spec-recon
	# macOS (Apple Silicon - ARM64)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/spec-recon
	# macOS (Intel - AMD64)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/spec-recon
	@echo "Release builds created in $(BUILD_DIR)"
