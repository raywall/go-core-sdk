package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/raywall/go-core-sdk/services/validation"
	validationtypes "github.com/raywall/go-core-sdk/services/validation/types"
)

// Order is the aggregate validated by the sample.
type Order struct {
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items" validate:"required,min=1"`
}

// Customer is the nested order customer.
type Customer struct {
	Document string `json:"document" validate:"required,len=11"`
}

// Item is one order line validated by the sample.
type Item struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"min=1"`
}

func main() {
	order := Order{
		Customer: Customer{Document: "123"},
		Items: []Item{
			{SKU: "", Quantity: 0},
		},
	}
	if err := run(context.Background(), os.Stdout, order); err != nil {
		log.Fatal(err)
	}
}

// OrderValidationUseCase validates an order and reports invalid fields.
type OrderValidationUseCase struct {
	Output io.Writer
}

func run(ctx context.Context, out io.Writer, order Order) error {
	return OrderValidationUseCase{Output: out}.Execute(ctx, order)
}

func (u OrderValidationUseCase) Execute(ctx context.Context, order Order) error {
	validator, err := validation.New()
	if err != nil {
		return err
	}

	if err := validator.Validate(ctx, order); err != nil {
		var validationErr *validationtypes.ValidationError
		if errors.As(err, &validationErr) {
			if _, err := fmt.Fprintf(u.Output, "invalidFields=%d\n", len(validationErr.Fields)); err != nil {
				return err
			}
			for _, field := range validationErr.Fields {
				if _, err := fmt.Fprintf(u.Output, "field=%s tag=%s\n", field.Namespace, field.Tag); err != nil {
					return err
				}
			}
			return nil
		}
		return err
	}

	_, err = fmt.Fprintln(u.Output, "order is valid")
	return err
}
