package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type total struct {
	Category  string `json:"category"`
	Quantity  int64  `json:"quantity"`
	CostMinor int64  `json:"cost_minor"`
}

func summarize(in io.Reader) ([]total, error) {
	r := csv.NewReader(in)
	r.FieldsPerRecord = 3
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	if header[0] != "category" || header[1] != "quantity" || header[2] != "unit_cost_minor" {
		return nil, fmt.Errorf("expected header category,quantity,unit_cost_minor")
	}
	groups := make(map[string]total)
	for record := 2; ; record++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", record, err)
		}
		name := strings.TrimSpace(row[0])
		q, qerr := strconv.ParseInt(row[1], 10, 64)
		cost, cerr := strconv.ParseInt(row[2], 10, 64)
		if name == "" || qerr != nil || cerr != nil || q <= 0 || cost < 0 {
			return nil, fmt.Errorf("record %d: need a category, positive quantity and nonnegative unit cost", record)
		}
		if cost > math.MaxInt64/q {
			return nil, fmt.Errorf("record %d: row cost overflow", record)
		}
		lineCost := q * cost
		g := groups[name]
		if g.Quantity > math.MaxInt64-q || g.CostMinor > math.MaxInt64-lineCost {
			return nil, fmt.Errorf("record %d: category total overflow", record)
		}
		g.Category = name
		g.Quantity += q
		g.CostMinor += lineCost
		groups[name] = g
	}
	result := make([]total, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Category < result[j].Category })
	return result, nil
}

func run(args []string, in io.Reader, out, diag io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(diag, "Usage: go-csvstats < items.csv")
		return 2
	}
	result, err := summarize(in)
	if err == nil {
		err = json.NewEncoder(out).Encode(result)
	}
	if err != nil {
		fmt.Fprintln(diag, err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
