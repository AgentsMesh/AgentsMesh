# Demo API

A simple Go HTTP API server for testing AgentsMesh development environment.

## Structure

```
├── main.go         # Entry point with HTTP handlers
├── main_test.go    # Unit tests
├── go.mod          # Go module definition
└── README.md
```

## Run

```bash
go run main.go
```

Server starts on `:8080` by default.

## API Endpoints

| Method | Path          | Description           |
|--------|---------------|-----------------------|
| GET    | /             | Health check          |
| GET    | /api/items    | List all items        |
| POST   | /api/items    | Create a new item     |
| GET    | /api/items/:id| Get item by ID        |

## Test

```bash
go test -v ./...
```

## Development Tasks

- Add DELETE endpoint for items
- Add pagination to list endpoint
- Add input validation
- Add persistent storage (SQLite)
