APP_NAME := git-safe
BUILD_DIR := build
BINARY := $(BUILD_DIR)/$(APP_NAME)
GO := go

.PHONY: help build test run clean

help:
	@echo "Targets:"
	@echo "  make build  - Build binary to $(BINARY)"
	@echo "  make test   - Run all tests"
	@echo "  make run    - Run app with ARGS='...'"
	@echo "  make clean  - Remove build artifacts"

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BINARY) .

test:
	$(GO) test ./...

run:
	$(GO) run . $(ARGS)

clean:
	rm -f $(BINARY)
