package main

import (
	"bytes"
	"strings"
	"testing"
)

const header = "category,quantity,unit_cost_minor\n"

func TestRunAggregation(t *testing.T) {
	var out, diag bytes.Buffer
	code := run(nil, strings.NewReader(header+"wood,2,125\n\"bolts, small\",3,10\n wood ,1,50\n"), &out, &diag)
	want := "[{\"category\":\"bolts, small\",\"quantity\":3,\"cost_minor\":30},{\"category\":\"wood\",\"quantity\":3,\"cost_minor\":300}]\n"
	if code != 0 || out.String() != want || diag.Len() != 0 {
		t.Fatalf("exit=%d output=%q error=%q", code, out.String(), diag.String())
	}
}

func TestRunRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"header", "wrong,quantity,unit_cost_minor\n", "expected header"},
		{"malformed", header + "wood,1\n", "record 2"},
		{"quantity", header + "wood,0,200\n", "positive quantity"},
		{"row overflow", header + "wood,2,9223372036854775807\n", "row cost overflow"},
		{"group overflow", header + "wood,1,9223372036854775807\nwood,1,1\n", "category total overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, diag bytes.Buffer
			code := run(nil, strings.NewReader(tc.input), &out, &diag)
			if code != 1 || out.Len() != 0 || !strings.Contains(diag.String(), tc.want) {
				t.Fatalf("exit=%d output=%q error=%q", code, out.String(), diag.String())
			}
		})
	}
}

func TestRunEmptyAndUsage(t *testing.T) {
	var out, diag bytes.Buffer
	if code := run(nil, strings.NewReader(header), &out, &diag); code != 0 || out.String() != "[]\n" {
		t.Fatalf("exit=%d output=%q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"items.csv"}, strings.NewReader(header), &out, &diag); code != 2 || out.Len() != 0 || !strings.Contains(diag.String(), "Usage:") {
		t.Fatalf("exit=%d output=%q error=%q", code, out.String(), diag.String())
	}
}
