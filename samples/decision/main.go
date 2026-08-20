package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/raywall/go-core-sdk/services/decision"
	decisiontypes "github.com/raywall/go-core-sdk/services/decision/types"
)

// Worker is the domain entity used by the margin rule.
type Worker struct {
	Active          bool  `json:"active"`
	AvailableMargin int64 `json:"availableMargin"`
}

// Proposal is the operation evaluated against the worker margin.
type Proposal struct {
	Amount int64 `json:"amount"`
}

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// MarginDecisionUseCase evaluates a proposal approval rule.
type MarginDecisionUseCase struct {
	Output io.Writer
}

func run(ctx context.Context, out io.Writer) error {
	return MarginDecisionUseCase{Output: out}.Execute(ctx)
}

func (u MarginDecisionUseCase) Execute(ctx context.Context) error {
	engine, err := decision.New()
	if err != nil {
		return err
	}

	result, err := engine.Evaluate(ctx, decisiontypes.EvaluationInput{
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
		return err
	}

	_, err = fmt.Fprintf(u.Output, "rule=%s allowed=%t cacheHit=%t\n", result.RuleName, result.Allowed, result.CacheHit)
	return err
}
