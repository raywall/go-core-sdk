// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/parser demonstrates DTO-to-entity conversion.
//
// This file is part of the Parser bounded context within the Parser service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/raywall/go-core-sdk/services/parser"
)

func main() {
	ctx := context.Background()
	dto := ProposalDTO{
		ID:             "proposal-123",
		Customer:       CustomerDTO{Document: "12345678901", Name: "Ana"},
		RequestedCents: 75000,
		Channel:        "mobile",
	}

	entity, err := parser.ParseAs[Proposal](ctx, dto)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("proposal=%s customer=%s amount=%d\n", entity.ID, entity.Customer.Document, entity.AmountCents)
}

// ProposalDTO is an inbound API payload with transport-oriented names.
type ProposalDTO struct {
	ID             string      `json:"id"`
	Customer       CustomerDTO `json:"customer"`
	RequestedCents int64       `json:"amountCents"`
	Channel        string      `json:"channel"`
}

// CustomerDTO is the customer data received by the API.
type CustomerDTO struct {
	Document string `json:"document"`
	Name     string `json:"name"`
}

// Proposal is a domain entity assembled from compatible JSON tags.
type Proposal struct {
	ID          string   `json:"id"`
	Customer    Customer `json:"customer"`
	AmountCents int64    `json:"amountCents"`
}

// Customer is the domain representation used by Proposal.
type Customer struct {
	Document string `json:"document"`
}
