package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunReportsValidationFields(t *testing.T) {
	t.Parallel()

	order := Order{
		Customer: Customer{Document: "123"},
		Items: []Item{
			{SKU: "", Quantity: 0},
		},
	}

	var out bytes.Buffer
	if err := run(context.Background(), &out, order); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"invalidFields=3", "field=Order.customer.document tag=len", "field=Order.items[0].sku tag=required"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
