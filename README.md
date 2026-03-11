# Todo CLI

A command-line todo manager built with Go, backed by MongoDB, with a gRPC client-server architecture.

## Prerequisites

- Go 1.25+
- MongoDB (local or Atlas)
- `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` (for regenerating protobuf code)

## Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/amharshit45/todos-cli-.git
   cd todos-cli-
   ```

2. Create a `.env` file in the project root:

   ```
   MONGO_URI=mongodb+srv://<user>:<password>@<cluster>/?appName=<app>
   MONGO_DB=todocli
   GRPC_ADDR=:50051
   ```

3. Install dependencies:

   ```bash
   go mod download
   ```

## Architecture

The application is split into a gRPC server and a CLI client:

```
CLI Client ──gRPC──▶ TodoService Server ──▶ MongoDB
```

- **Server** (`cmd/server`): Hosts the `TodoService` gRPC service backed by MongoDB.
- **Client** (`cmd/client`): Interactive CLI that sends requests to the server over gRPC.

## Build & Run

Build both binaries:

```bash
make build
```

Start the server (requires MongoDB):

```bash
make run-server
```

In another terminal, start the client:

```bash
make run-client
```

Or run directly with `go run`:

```bash
go run ./cmd/server
# in another terminal
go run ./cmd/client
```

## Usage

The CLI presents an interactive menu:

```
===== Todo CLI =====
1. Add a todo
2. List todos
3. Delete a todo
4. Mark as completed
5. Mark as incomplete
6. Edit a todo
7. Exit
====================
```

## Configuration

| Variable    | Description                | Default          |
|-------------|----------------------------|------------------|
| `MONGO_URI` | MongoDB connection string  | *(required)*     |
| `MONGO_DB`  | MongoDB database name      | *(required)*     |
| `GRPC_ADDR` | gRPC listen/connect address| `:50051`         |

The server uses `GRPC_ADDR` as the listen address; the client uses it as the dial target (defaults to `localhost:50051`).

## Project Structure

```
.
├── cmd/
│   ├── server/main.go           # gRPC server entry point
│   └── client/main.go           # CLI client entry point
├── proto/todo/v1/todo.proto     # Protobuf service definition
├── gen/todopb/                  # Generated protobuf + gRPC Go code
├── server/
│   ├── grpc.go                  # gRPC service implementation
│   └── grpc_test.go             # Server tests (bufconn + mock storage)
├── grpcclient/
│   └── client.go                # gRPC client implementing todo.Storage
├── cli/
│   ├── cli.go                   # Interactive CLI (unchanged)
│   └── cli_test.go              # CLI tests (mock storage)
├── todo/
│   ├── model.go                 # Todo struct and validation
│   ├── storage.go               # Storage interface
│   └── errors.go                # Domain errors
├── storage/
│   ├── mongo.go                 # MongoDB storage implementation
│   └── mongo_test.go            # MongoDB integration tests
├── Makefile                     # Build, run, test, proto targets
└── .env                         # Config (not committed)
```

## Regenerating Protobuf Code

```bash
make proto
```

Requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` to be installed.

## Testing

```bash
make test
```

## Clean

```bash
make clean
```
