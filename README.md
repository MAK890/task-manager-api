# Task Manager API

A small Go REST API that stores tasks in MySQL and uses Redis as a one-minute
read cache. The project is split into packages so HTTP, business rules,
persistence, caching, configuration, and server lifecycle can change
independently.

## Requirements

- Go 1.26 or newer
- MySQL on `127.0.0.1:3306` by default
- Redis on `127.0.0.1:6379` by default

## Configuration

Every setting can be supplied through the environment. The non-secret settings
have development defaults; the MySQL password intentionally does not.

```bash
export MYSQL_PASSWORD='your-password'
go run .
```

See [.env.example](.env.example) for every supported variable. This project
does not automatically load `.env`; export the values in your shell or use your
process/container environment.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Check that the HTTP process is running |
| `POST` | `/tasks` | Create a task |
| `GET` | `/tasks` | List tasks |
| `GET` | `/tasks/{id}` | Fetch one task |
| `PUT` | `/tasks/{id}` | Replace a task |
| `DELETE` | `/tasks/{id}` | Delete a task |

Example request:

```bash
curl -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Learn Go","description":"Trace one request","priority":"high"}'
```

## Package map

```text
main.go                 creates and connects every dependency
internal/config         reads environment settings
internal/model          defines Task
internal/database       opens the MySQL connection
internal/cache          implements JSON caching with Redis
internal/repository     runs task SQL queries
internal/service        owns validation, workflow, and cache coordination
internal/httpapi        maps HTTP requests and responses
internal/server         starts and gracefully stops net/http
```

The detailed explanation and suggested study order are in
[docs/LEARNING_GUIDE.md](docs/LEARNING_GUIDE.md).

## Verification

```bash
go test ./...
go vet ./...
go build -buildvcs=false ./...
```

`-buildvcs=false` is useful when this directory is not inside a valid Git
checkout.
