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
	"io"
	"log"
	"os"

	"github.com/raywall/go-core-sdk/services/environment"
)

func main() {
	values := map[string]string{
		"APP_SERVICE_NAME": "orders-worker",
	}
	if err := run(context.Background(), os.Stdout, func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}); err != nil {
		log.Fatal(err)
	}
}

// EnvironmentUseCase loads required and optional runtime settings.
type EnvironmentUseCase struct {
	Lookup environment.LookupFunc
	Output io.Writer
}

func run(ctx context.Context, out io.Writer, lookup environment.LookupFunc) error {
	return EnvironmentUseCase{Lookup: lookup, Output: out}.Execute(ctx)
}

func (u EnvironmentUseCase) Execute(ctx context.Context) error {
	env, err := environment.New(environment.WithLookupFunc(func(name string) (string, bool) {
		return u.Lookup(name)
	}))
	if err != nil {
		return err
	}

	serviceName, err := env.Get(ctx, "APP_SERVICE_NAME")
	if err != nil {
		return err
	}
	environmentName, err := env.GetDefault(ctx, "APP_ENVIRONMENT", "local")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(u.Output, "service=%s environment=%s\n", serviceName, environmentName)
	return err
}
