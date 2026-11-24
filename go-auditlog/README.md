# Go Audit Log

A small standard-library CLI for appending fictional application events to a local JSON Lines file and reading them back in file order.

## Try it

Requires Go 1.22 or later.

```sh
go run . append -file demo.jsonl -action created -subject fictional-job-7
go run . append -file demo.jsonl -action finished -subject fictional-job-7
go run . read -file demo.jsonl
```

Each record contains UTC `time`, `action`, and `subject`. A successful append produces no terminal output. Reading emits one record per line and stops at the first invalid record, with its line number in the error. Records before that line may already have been printed. The compiled CLI exits with 0 for success, 1 for an I/O or data error, and 2 for incorrect usage.

## Scope and limitations

Use one writer at a time. Append assumes an existing log is valid JSON Lines and ends with a newline; it does not validate or repair a truncated tail. The CLI has no lock coordination, crash-durability guarantee, rotation, access control, or tamper evidence. The filesystem does not prevent another program from editing the file. Keep real user data and credentials out of this learning project.

```sh
go test ./...
go vet ./...
```

Exercise: add a `-action` read filter without overwriting or skipping validation of the original records. Write a test that catches a corrupted record that occurs after a matching event.

Built with AI assistance as a learning project.
