package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/raywall/go-core-sdk/services/selector"
	selectortypes "github.com/raywall/go-core-sdk/services/selector/types"
)

type Installment struct {
	Number      int       `json:"number"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"dueDate"`
	AmountCents int64     `json:"amountCents"`
}

func main() {
	installments := []Installment{
		{Number: 3, Status: "OPEN", DueDate: mustDate("2026-03-01"), AmountCents: 10_000},
		{Number: 1, Status: "OPEN", DueDate: mustDate("2026-01-01"), AmountCents: 10_000},
		{Number: 2, Status: "OPEN", DueDate: mustDate("2026-02-01"), AmountCents: 10_000},
	}

	ordered, result, err := selector.SortAndSelect(context.Background(), installments,
		selectortypes.SortConfig{
			Path:      "dueDate",
			Kind:      selectortypes.KindTime,
			Direction: selectortypes.Ascending,
		},
		selectortypes.SelectionConfig{
			AmountPath:      "amountCents",
			AvailableAmount: 25_000,
			Mode:            selectortypes.ModePartial,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("firstDue=%s payments=%d applied=%d remaining=%d\n",
		ordered[0].DueDate.Format(time.DateOnly),
		len(result.Payments),
		result.TotalAppliedAmount,
		result.RemainingAmount,
	)
	for _, payment := range result.Payments {
		fmt.Printf("installment=%d applied=%d partial=%t\n", payment.Item.Number, payment.AppliedAmount, payment.Partial)
	}
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
