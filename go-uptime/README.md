# Go Uptime Checker

A small CLI that checks only the HTTP(S) URLs you give it. It reports each URL in input order even though requests run concurrently. It uses only the Go standard library.

## Run

Requires Go 1.22 or newer. From this directory:

```sh
go run . -timeout=3s -workers=4 https://example.com
```

Example output:

```text
OK "https://example.com" HTTP 200
```

The CLI accepts up to 64 URLs per invocation and 1–16 concurrent workers. Set flags before URLs. Use endpoints you own or are permitted to check. Do not put secrets in URLs: the provided URLs are displayed in the terminal.

## What it measures

- One explicit GET per supplied URL, with no redirect following or body processing.
- HTTP 2xx counts as success; redirects, other statuses and request failures do not.
- This is a point-in-time check, not continuous monitoring. It does not verify response content or long-term uptime.
- The binary exits with 0 if all URLs pass, 1 if any check fails, or 2 for invalid arguments. Use the built binary to observe the exact exit code; `go run` reports nonzero program exit codes as errors.

```sh
go build -o go-uptime .
go test ./...
go vet ./...
```

The tests use local HTTP servers only. As a practice exercise, add a JSON output option without changing the exit codes or result order.

Built with AI assistance as a learning project.
