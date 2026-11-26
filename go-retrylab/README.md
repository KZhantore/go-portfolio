# Retry Lab

A small Go CLI that traces an exponential-backoff policy on a virtual clock. Compare retry limits and wait budgets without sleeping or calling any service. No third-party packages.

## Run

Requires Go 1.22 or later. From this directory:

```sh
go run . -script retry,retry,ok -base-ms 100 -cap-ms 250 -attempts 4 -budget-ms 500
```

```text
attempt 1 at 0ms: retry
attempt 2 at 100ms: retry
attempt 3 at 300ms: ok
stop: success
```

The first attempt is immediate and counts toward the total. After a retryable outcome, the delay doubles until it reaches the cap. A wait that lands exactly on the budget is allowed. A permanent failure stops immediately. After a retryable outcome, the simulator checks the attempt limit, then the wait budget, then whether the script has a next outcome. A valid simulation exits with code 0 even when its scripted operation fails; code 2 indicates invalid CLI input and code 1 indicates a write failure.

## Limits and next step

The script is local and deterministic. There is no networking, jitter, real execution time, or recovery after an exhausted script. The budget is for waits only and is not a real request deadline.

**Practice:** Add optional, seeded jitter to the next delay. Test that results are repeatable for the same seed, that no delay exceeds the configured cap, and that the budget is still enforced after jitter.

```sh
go test ./...
go vet ./...
```

Built with AI assistance as a backend portfolio exercise.
