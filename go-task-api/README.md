# Go Task API

A small REST service for tracking tasks, built with Go's standard library. It runs locally and keeps tasks after a restart without requiring a database.

## Run

Requires Go 1.22 or newer.

```sh
go run .
```

The service listens on `127.0.0.1:8080`. By default tasks are stored in `tasks.json`. Set `TASKS_FILE` to use a different file in an existing directory.

## Try it

```sh
curl -i -X POST http://127.0.0.1:8080/tasks \
  -H 'Content-Type: application/json' -d '{"title":"Learn Go"}'
curl http://127.0.0.1:8080/tasks
curl http://127.0.0.1:8080/tasks/1
curl -i -X PUT http://127.0.0.1:8080/tasks/1 \
  -H 'Content-Type: application/json' -d '{"title":"Learn Go","done":true}'
curl -i -X DELETE http://127.0.0.1:8080/tasks/1
```

Replace `1` with the ID returned by your POST if the store already contains tasks.

| Method | Path | Result |
| --- | --- | --- |
| GET | /tasks | List tasks (200) |
| POST | /tasks | Create a task (201) |
| GET | /tasks/{id} | Read a task (200) |
| PUT | /tasks/{id} | Replace task fields (200) |
| DELETE | /tasks/{id} | Delete a task (204) |

Titles are trimmed and must contain 1–120 Unicode code points. PUT replaces both fields: omitting `done` sets it to false. Invalid inputs return 400 (413 for payloads above 8 KiB); missing tasks return 404.

## How it works

`main.go` defines routes and validates requests. `store.go` protects concurrent access using a mutex. For every write it stages a copy, writes that copy to a temporary file next to the destination, and renames it into place before updating memory. A failed write leaves memory unchanged. Run a single process per data file. This local learning project does not include authentication.

```sh
go test ./...
go vet ./...
```

A useful next exercise: add a `?done=true` filter to the list endpoint and test both valid and invalid filter values.
