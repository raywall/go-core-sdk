package main

import (
	"context"
	"fmt"
	"log"

	"github.com/raywall/go-core-sdk/services/decision"
	decisiontypes "github.com/raywall/go-core-sdk/services/decision/types"
)

type Worker struct {
	Active          bool  `json:"active"`
	AvailableMargin int64 `json:"availableMargin"`
}

type Proposal struct {
	Amount int64 `json:"amount"`
}

func main() {
	engine, err := decision.New()
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine.Evaluate(context.Background(), decisiontypes.EvaluationInput{
		Rule: decisiontypes.Rule{
			Name:        "margin-approved",
			Description: "Worker must be active and have enough available margin.",
			Expression:  "worker.active && proposal.amount <= worker.availableMargin",
		},
		Entities: map[string]any{
			"worker":   Worker{Active: true, AvailableMargin: 100_000},
			"proposal": Proposal{Amount: 75_000},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("rule=%s allowed=%t cacheHit=%t\n", result.RuleName, result.Allowed, result.CacheHit)
}
