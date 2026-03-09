# Todo CLI

A command-line todo manager built with Go, backed by MongoDB.

## Prerequisites

- Go 1.25+
- MongoDB (local or Atlas)

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
   ```

3. Install dependencies:

   ```bash
   go mod download
   ```

## Build & Run

```bash
make build
make run
```

Or run directly:

```bash
go run main.go
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

## Project Structure

```
.
├── main.go              # CLI entry point and REPL loop
├── todo/
│   ├── model.go         # Todo struct definition
│   └── storage.go       # Storage interface
├── storage/
│   ├── json.go          # JSON file-based storage implementation
│   └── mongo.go         # MongoDB storage implementation
├── Makefile             # Build, run, test, clean targets
└── .env                 # MongoDB connection config (not committed)
```

## Testing

```bash
make test
```

## Clean

```bash
make clean
```
