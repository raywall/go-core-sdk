// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/environment demonstrates environment variable loading.
//
// This file is part of the Environment bounded context within the Environment
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/raywall/go-core-sdk/services/environment"
)

func main() {
	ctx := context.Background()
	values := map[string]string{
		"APP_SERVICE_NAME": "orders-worker",
	}

	env, err := environment.New(environment.WithLookupFunc(func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}))
	if err != nil {
		log.Fatal(err)
	}

	serviceName, err := env.Get(ctx, "APP_SERVICE_NAME")
	if err != nil {
		log.Fatal(err)
	}
	environmentName, err := env.GetDefault(ctx, "APP_ENVIRONMENT", "local")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("service=%s environment=%s\n", serviceName, environmentName)
}
