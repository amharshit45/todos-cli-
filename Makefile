APP_NAME := todos-cli
BINARY_DIR := bin

.PHONY: build build-server build-client run-server run-client test vet clean proto

proto:
	protoc \
		--go_out=gen/todopb --go_opt=module=github.com/amharshit45/todos-cli-/gen/todopb \
		--go-grpc_out=gen/todopb --go-grpc_opt=module=github.com/amharshit45/todos-cli-/gen/todopb \
		proto/todo/v1/todo.proto

build-server:
	go build -o $(BINARY_DIR)/$(APP_NAME)-server ./cmd/server

build-client:
	go build -o $(BINARY_DIR)/$(APP_NAME)-client ./cmd/client

build: build-server build-client

run-server: build-server
	$(BINARY_DIR)/$(APP_NAME)-server

run-client: build-client
	$(BINARY_DIR)/$(APP_NAME)-client

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BINARY_DIR)
