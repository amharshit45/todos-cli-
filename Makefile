APP_NAME := todos-cli
BINARY_DIR := bin

.PHONY: build run test vet clean

build:
	go build -o $(BINARY_DIR)/$(APP_NAME)

run:
	$(BINARY_DIR)/$(APP_NAME)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BINARY_DIR)
