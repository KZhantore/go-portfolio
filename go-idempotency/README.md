# Idempotent Orders

A local Go HTTP service that shows how safely repeating a request can avoid duplicate work. The example uses fictional notebook orders and no external services.

## Run

Use Go 1.22+ and run `go run .` from this directory. In another terminal, run the same request twice:

```sh
curl -i http://127.0.0.1:8080/orders \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-1' \
  -d '{"item":"notebook","units":2}'
```

Both responses return `201 Created` and `{"id":1,"item":"notebook","units":2}`. The replay also returns `Idempotency-Replayed: true`. With the same key but different order fields, the API returns `409 Conflict`. Field order and surrounding spaces in the item name don't affect whether orders match. Invalid requests don't reserve keys.

The mutex protects the lookup and insert together, so two simultaneous requests can't create two orders with the same key. Request bodies are capped at 4 KiB and the ledger at 1,000 orders. A full ledger rejects new keys with `503` but still replays stored results.

## Limits and practice

This is a single-process, loopback-only demo. It does not authenticate callers, expire keys, persist data across restarts, or coordinate with external side effects.

**Practice:** Design a persistent version that namespaces keys by caller and stores the order and its idempotency record in a single database transaction. Explain what the client should receive if the order was committed but the first response never arrived.

```sh
go test -race ./...
go vet ./...
```

Built with AI assistance as a backend portfolio exercise.
