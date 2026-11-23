# Go TTL Cache

A small, generic, thread-safe cache using only Go's standard library. A fixed capacity limits how many keys can be stored. Each key has its own time to live (TTL).

## Run the demo

Requires Go 1.22 or later:

```sh
go run ./cmd/demo
```

```text
Before expiry: Go (hit: true)
At expiry: hit: false
```

The demo advances a fake clock rather than sleeping. Use `New[string](100, nil)` with the real clock. Call `Set(key, value, ttl)` to insert or replace a value, and `Get(key)` to retrieve a live value. A read makes the key recently used, but does not extend its TTL.

When full, the cache first tries to reclaim an expired entry and otherwise evicts the least recently used key. A lookup removes an expired entry. There is no background goroutine or persistence. A full insert may scan the bounded list to reclaim expired entries. Returned pointer-like values are not cloned: callers must handle synchronization of the values themselves.

## Verify

```sh
go test ./...
go test -race ./...
go vet ./...
```

Exercise: add a `Delete(key)` method and test that deleting a key immediately frees capacity without evicting another key.

Built with AI assistance as a learning project.
