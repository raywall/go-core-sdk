// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// lambda implements S3 event normalization.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/raywall/go-core-sdk/handlers"
)

// HandleS3 normalizes an AWS Lambda S3 event and invokes the processor.
func (r *Runner) HandleS3(ctx context.Context, event events.S3Event) (handlers.Result, error) {
	return r.processor.Process(ctx, NormalizeS3Event(event))
}

// NormalizeS3Event converts an AWS Lambda S3 event to handlers.Event.
func NormalizeS3Event(event events.S3Event) handlers.Event {
	records := make([]handlers.Record, 0, len(event.Records))
	for _, record := range event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key
		records = append(records, handlers.Record{
			ID:     fmt.Sprintf("%s/%s", bucket, key),
			Source: "aws:s3",
			Metadata: map[string]string{
				"eventName": record.EventName,
				"region":    record.AWSRegion,
			},
			S3: &handlers.S3Object{
				Bucket: bucket,
				Key:    key,
				ETag:   record.S3.Object.ETag,
				Size:   record.S3.Object.Size,
			},
		})
	}
	return handlers.Event{
		Runtime: handlers.RuntimeLambda,
		Trigger: handlers.TriggerS3,
		Records: records,
	}
}
