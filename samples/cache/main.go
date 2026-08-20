package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/raywall/go-core-sdk/services/cache"
)

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
	if err := store.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer store.Stop()

	if err := store.Add(ctx, "customer-123", Customer{ID: "customer-123", Name: "Ana"}); err != nil {
		log.Fatal(err)
	}

	customer, found, err := store.Get(ctx, "customer-123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("found=%t customer=%+v\n", found, customer)
}
