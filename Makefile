APP_NAME := todos-cli
BINARY_DIR := bin

.PHONY: build run test clean

build:
	go build -o $(BINARY_DIR)/$(APP_NAME)

run:
	$(BINARY_DIR)/$(APP_NAME)

test:
	go test ./...

clean:
	rm -rf $(BINARY_DIR)
