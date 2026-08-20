package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/services/cache"
)

// Customer is the entity cached by the sample use case.
type Customer struct {
	ID   string
	Name string
}

func main() {
	ctx := context.Background()
	store, err := cache.New[Customer](cache.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx, CustomerUseCase{
		Store:  store,
		Output: os.Stdout,
	}); err != nil {
		log.Fatal(err)
	}
}

type customerStore interface {
	Start(context.Context) error
	Stop()
	Add(context.Context, string, Customer, ...time.Duration) error
	Get(context.Context, string) (Customer, bool, error)
}

// CustomerUseCase stores and retrieves a customer through an injected cache port.
type CustomerUseCase struct {
	Store  customerStore
	Output io.Writer
}

func run(ctx context.Context, useCase CustomerUseCase) error {
	return useCase.Execute(ctx)
}

func (u CustomerUseCase) Execute(ctx context.Context) error {
	if err := u.Store.Start(ctx); err != nil {
		return err
	}
	defer u.Store.Stop()

	if err := u.Store.Add(ctx, "customer-123", Customer{ID: "customer-123", Name: "Ana"}); err != nil {
		return err
	}

	customer, found, err := u.Store.Get(ctx, "customer-123")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(u.Output, "found=%t customer=%+v\n", found, customer)
	return err
}
