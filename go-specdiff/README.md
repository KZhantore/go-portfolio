# Go SpecDiff

A standard-library CLI for comparing two JSON snapshots by stable item ID. This learning project uses fictional examples.

## Run

Requires Go 1.22 or later:

```sh
go run . examples/old.json examples/new.json
```

```text
Revision "demo-r1" -> "demo-r2"
QUANTITY CHANGED "board": 12 -> 15 "pc" (+3)
UNIT CHANGED "cord": 10 "m" -> 1000 "cm" (no numeric comparison)
ADDED "screw": 20 "pc"
REMOVED "tape": 1 "roll"
```

Each snapshot needs a nonblank revision and an items array. Each item requires a unique, nonblank ID, a nonblank unit, and a nonnegative whole-number quantity. IDs and units are compared exactly and case-sensitively. The report sorts changes by ID, regardless of input ordering. Changed units are flagged for review without assuming conversion factors.

Files are limited to 1 MiB each. The program does not read spreadsheets, normalize units, or infer missing items. It compares quantities, not prices, and is not an official procurement tool. The built CLI exits 0 for a successful comparison (including changes), 1 for an unreadable or invalid snapshot, and 2 for incorrect usage.

```sh
go test ./...
go vet ./...
```

Try adding a JSON output mode and a test that confirms input order does not affect the report.

Built with AI assistance as a learning project.
