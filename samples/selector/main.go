package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/services/selector"
	selectortypes "github.com/raywall/go-core-sdk/services/selector/types"
)

// Installment is the domain item sorted and selected by the sample.
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
	if err := run(context.Background(), os.Stdout, installments, 25_000); err != nil {
		log.Fatal(err)
	}
}

// InstallmentSelectionUseCase orders installments and applies an available amount.
type InstallmentSelectionUseCase struct {
	Output io.Writer
}

func run(ctx context.Context, out io.Writer, installments []Installment, availableAmount int64) error {
	return InstallmentSelectionUseCase{Output: out}.Execute(ctx, installments, availableAmount)
}

func (u InstallmentSelectionUseCase) Execute(ctx context.Context, installments []Installment, availableAmount int64) error {
	ordered, result, err := selector.SortAndSelect(ctx, installments,
		selectortypes.SortConfig{
			Path:      "dueDate",
			Kind:      selectortypes.KindTime,
			Direction: selectortypes.Ascending,
		},
		selectortypes.SelectionConfig{
			AmountPath:      "amountCents",
			AvailableAmount: availableAmount,
			Mode:            selectortypes.ModePartial,
		},
	)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(u.Output, "firstDue=%s payments=%d applied=%d remaining=%d\n",
		ordered[0].DueDate.Format(time.DateOnly),
		len(result.Payments),
		result.TotalAppliedAmount,
		result.RemainingAmount,
	); err != nil {
		return err
	}
	for _, payment := range result.Payments {
		if _, err := fmt.Fprintf(u.Output, "installment=%d applied=%d partial=%t\n", payment.Item.Number, payment.AppliedAmount, payment.Partial); err != nil {
			return err
		}
	}
	return nil
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
