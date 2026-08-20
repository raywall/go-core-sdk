// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements a facade for outbound REST and AWS integrations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package consumer provides a small integration facade for microservices.
//
// The service supports REST calls with custom headers and optional token
// injection, plus convenience methods for common DynamoDB, S3, Secrets Manager and SQS
// operations. AWS clients can be supplied explicitly for tests, tracing or
// custom transports; otherwise they are created lazily from the AWS SDK default
// configuration chain.
//
// Usage:
//
//	client, err := consumer.New(consumer.Config{})
//	if err != nil {
//		return err
//	}
//	response, err := client.REST(http.MethodPost, "https://api.example.com/orders").
//		WithHeader("X-App", "orders-api").
//		WithBody(order).
//		WithToken().
//		Do(ctx)
//
// Thread safety: Consumer is safe for concurrent use. Client construction is
// guarded internally and no request body or token values are logged.
package consumer
