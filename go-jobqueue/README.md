# Go Job Queue

A standard-library demo of a bounded, in-memory background job queue. A fixed pool of workers processes accepted jobs, while callers can detect backpressure without waiting for a free slot.

## Try it

Requires Go 1.22 or newer:

```sh
go run . -workers=2 -jobs=3
```

```text
Job 1: 1
Job 2: 4
Job 3: 9
```

The demo computes squares concurrently and prints the results in submission order once every job finishes. Its buffer is sized for the batch. In the reusable queue itself, capacity counts waiting jobs, in addition to the jobs already running. `TrySubmit` returns `ErrFull` if there is no waiting slot, or `ErrClosed` after shutdown starts. `Shutdown` stops admissions and drains accepted jobs; call it outside worker jobs.

## Verify

```sh
go test ./...
go test -race ./...
go vet ./...
```

This is a local teaching example, not a durable job system: there is no persistence, retry, cancellation, or panic recovery. A job that never returns can prevent shutdown from completing.

Try adding a context-aware shutdown wait and write a test that demonstrates what happens when a worker is blocked.

Built with AI assistance as a learning project.
