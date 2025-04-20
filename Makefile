.PHONY: build run migrate test clean

GO=go
BIN_DIR=bin
APP_NAME=tvheadend
MAIN_PKG=cmd/tvheadend/main.go

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PKG)

run: build
	$(BIN_DIR)/$(APP_NAME)

test:
	$(GO) test -v ./...

clean:
	rm -rf $(BIN_DIR)

migrate-up:
	$(GO) run cmd/migrate/main.go up

migrate-down:
	$(GO) run cmd/migrate/main.go down