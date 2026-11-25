# Go CSV Stats

A small CLI that reads a CSV of fictional stock items and produces ordered JSON totals by category. It uses integer minor units so calculations are exact and rejects values that overflow.

## Run

Requires Go 1.22 or newer, with no external dependencies.

```sh
printf 'category,quantity,unit_cost_minor\nwood,2,125\nbolts,3,10\nwood,1,50\n' | go run .
```

```json
[{"category":"bolts","quantity":3,"cost_minor":30},{"category":"wood","quantity":3,"cost_minor":300}]
```

The header must match exactly. Quantities must be positive and unit costs nonnegative. Quoted fields follow Go's CSV parser. Category names are trimmed and case sensitive. Invalid input is reported at the record that fails, with no JSON output.

## Checks

```sh
gofmt -l .
go test ./...
go vet ./...
```

## Limitations and exercise

Assumes one currency and does not handle tax, discounts, or refunds. It reads one record at a time but keeps one total per unique category in memory. It does not impose hard input size limits; use it with trusted local data.

Exercise: add a JSON field for the number of records in each category. Test that one record with quantity 10 increases the record count by 1 rather than 10.

Built with AI assistance as a learning project.
