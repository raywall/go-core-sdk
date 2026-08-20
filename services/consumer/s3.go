// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements S3 convenience operations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// PutS3 uploads an object to S3.
func (c *Consumer) PutS3(ctx context.Context, input types.S3PutInput) (types.S3PutOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.s3Client(ctx)
	if err != nil {
		return types.S3PutOutput{}, types.S3Error{Operation: "load_config", Err: err}
	}
	body, err := encodeS3Body(input.Body)
	if err != nil {
		return types.S3PutOutput{}, types.S3Error{Operation: "encode_body", Err: err}
	}
	request := &s3.PutObjectInput{
		Bucket:      aws.String(strings.TrimSpace(input.Bucket)),
		Key:         aws.String(strings.TrimSpace(input.Key)),
		Body:        body,
		ContentType: optionalString(input.ContentType),
		Metadata:    input.Metadata,
	}
	c.logger.InfoContext(ctx, "consumer_s3_put_started", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key))
	response, err := client.PutObject(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_s3_put_failed", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key), "error", err)
		return types.S3PutOutput{}, types.S3Error{Operation: "put_object", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_s3_put_completed", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key))
	return types.S3PutOutput{ETag: aws.ToString(response.ETag)}, nil
}

// GetS3 downloads an object from S3 and returns its full body.
func (c *Consumer) GetS3(ctx context.Context, input types.S3GetInput) (types.S3GetOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.s3Client(ctx)
	if err != nil {
		return types.S3GetOutput{}, types.S3Error{Operation: "load_config", Err: err}
	}
	request := &s3.GetObjectInput{
		Bucket: aws.String(strings.TrimSpace(input.Bucket)),
		Key:    aws.String(strings.TrimSpace(input.Key)),
	}
	c.logger.InfoContext(ctx, "consumer_s3_get_started", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key))
	response, err := client.GetObject(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_s3_get_failed", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key), "error", err)
		return types.S3GetOutput{}, types.S3Error{Operation: "get_object", Err: err}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return types.S3GetOutput{}, types.S3Error{Operation: "read_object", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_s3_get_completed", "bucket", strings.TrimSpace(input.Bucket), "key", strings.TrimSpace(input.Key), "bytes", len(body))
	return types.S3GetOutput{
		Body:        body,
		ContentType: aws.ToString(response.ContentType),
		Metadata:    response.Metadata,
	}, nil
}

func encodeS3Body(body any) (io.Reader, error) {
	switch value := body.(type) {
	case nil:
		return bytes.NewReader(nil), nil
	case io.Reader:
		return value, nil
	case []byte:
		return bytes.NewReader(value), nil
	case string:
		return strings.NewReader(value), nil
	default:
		return nil, types.InvalidConfigError{Field: "S3PutInput.Body", Reason: "must be nil, []byte, string or io.Reader"}
	}
}
