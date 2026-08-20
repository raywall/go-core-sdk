package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/raywall/go-core-sdk/services/validation"
	validationtypes "github.com/raywall/go-core-sdk/services/validation/types"
)

type Order struct {
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items" validate:"required,min=1"`
}

type Customer struct {
	Document string `json:"document" validate:"required,len=11"`
}

type Item struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"min=1"`
}

func main() {
	validator, err := validation.New()
	if err != nil {
		log.Fatal(err)
	}

	order := Order{
		Customer: Customer{Document: "123"},
		Items: []Item{
			{SKU: "", Quantity: 0},
		},
	}

	if err := validator.Validate(context.Background(), order); err != nil {
		var validationErr *validationtypes.ValidationError
		if errors.As(err, &validationErr) {
			fmt.Printf("invalidFields=%d\n", len(validationErr.Fields))
			for _, field := range validationErr.Fields {
				fmt.Printf("field=%s tag=%s\n", field.Namespace, field.Tag)
			}
			return
		}
		log.Fatal(err)
	}

	fmt.Println("order is valid")
}
