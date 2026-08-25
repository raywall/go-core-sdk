// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements adapters for the official Hazelcast client.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"context"

	hz "github.com/hazelcast/hazelcast-go-client"
)

type provider struct {
	client *hz.Client
}

func startProvider(ctx context.Context, config hz.Config) (MapProvider, error) {
	client, err := hz.StartNewClientWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return provider{client: client}, nil
}

func (p provider) GetMap(ctx context.Context, name string) (Map, error) {
	return p.client.GetMap(ctx, name)
}

func (p provider) Shutdown(ctx context.Context) error {
	return p.client.Shutdown(ctx)
}
