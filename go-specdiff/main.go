package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

type item struct {
	ID       string `json:"id"`
	Quantity *int64 `json:"quantity"`
	Unit     string `json:"unit"`
}

type snapshot struct {
	Revision string `json:"revision"`
	Items    []item `json:"items"`
}

func load(r io.Reader) (snapshot, error) {
	var s snapshot
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return s, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return s, errors.New("expected exactly one JSON object")
	}
	if strings.TrimSpace(s.Revision) == "" || s.Items == nil {
		return s, errors.New("revision and items array are required")
	}
	seen := make(map[string]bool, len(s.Items))
	for n, i := range s.Items {
		if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.Unit) == "" || i.Quantity == nil || *i.Quantity < 0 {
			return s, fmt.Errorf("item %d: nonblank id and unit and a nonnegative integer quantity are required", n+1)
		}
		if seen[i.ID] {
			return s, fmt.Errorf("duplicate item ID %q", i.ID)
		}
		seen[i.ID] = true
	}
	return s, nil
}

func diff(old, new snapshot, w io.Writer) {
	before := make(map[string]item, len(old.Items))
	after := make(map[string]item, len(new.Items))
	ids := make([]string, 0, len(old.Items)+len(new.Items))
	for _, i := range old.Items {
		before[i.ID] = i
		ids = append(ids, i.ID)
	}
	for _, i := range new.Items {
		after[i.ID] = i
		if _, exists := before[i.ID]; !exists {
			ids = append(ids, i.ID)
		}
	}
	slices.Sort(ids)
	fmt.Fprintf(w, "Revision %q -> %q\n", old.Revision, new.Revision)
	changes := 0
	for _, id := range ids {
		a, hasA := before[id]
		b, hasB := after[id]
		switch {
		case !hasA:
			fmt.Fprintf(w, "ADDED %q: %d %q\n", id, *b.Quantity, b.Unit)
		case !hasB:
			fmt.Fprintf(w, "REMOVED %q: %d %q\n", id, *a.Quantity, a.Unit)
		case a.Unit != b.Unit:
			fmt.Fprintf(w, "UNIT CHANGED %q: %d %q -> %d %q (no numeric comparison)\n", id, *a.Quantity, a.Unit, *b.Quantity, b.Unit)
		case *a.Quantity != *b.Quantity:
			fmt.Fprintf(w, "QUANTITY CHANGED %q: %d -> %d %q (%+d)\n", id, *a.Quantity, *b.Quantity, a.Unit, *b.Quantity-*a.Quantity)
		default:
			continue
		}
		changes++
	}
	if changes == 0 {
		fmt.Fprintln(w, "No item changes.")
	}
}

func loadFile(path string) (snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return snapshot{}, err
	}
	defer f.Close()
	// Bound memory use, including when the path points to a stream.
	const max = 1 << 20
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return snapshot{}, err
	}
	if len(data) > max {
		return snapshot{}, errors.New("snapshot exceeds 1 MiB")
	}
	return load(strings.NewReader(string(data)))
}

func run(args []string, out, diagnostics io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(out, "Usage: go-specdiff OLD.json NEW.json")
		return 0
	}
	if len(args) != 2 {
		fmt.Fprintln(diagnostics, "Usage: go-specdiff OLD.json NEW.json")
		return 2
	}
	old, err := loadFile(args[0])
	if err != nil {
		fmt.Fprintf(diagnostics, "Old snapshot: %v\n", err)
		return 1
	}
	new, err := loadFile(args[1])
	if err != nil {
		fmt.Fprintf(diagnostics, "New snapshot: %v\n", err)
		return 1
	}
	diff(old, new, out)
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
