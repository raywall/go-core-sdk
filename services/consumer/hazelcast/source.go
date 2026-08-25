// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements Hazelcast configuration source loading.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"context"
	"os"

	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

type defaultSourceLoader struct {
	awsClient *consumeraws.Client
	logger    logger
}

type logger interface {
	InfoContext(context.Context, string, ...any)
	ErrorContext(context.Context, string, ...any)
}

func (l defaultSourceLoader) Load(ctx context.Context, source Source) ([]byte, error) {
	if err := validateSource(source); err != nil {
		return nil, err
	}
	switch source.Kind {
	case SourceDefault:
		return nil, nil
	case SourceInline:
		return append([]byte(nil), source.Data...), nil
	case SourceFile:
		l.logger.InfoContext(ctx, "consumer_hazelcast_config_file_read_started", "path", source.Path)
		data, err := os.ReadFile(source.Path)
		if err != nil {
			l.logger.ErrorContext(ctx, "consumer_hazelcast_config_file_read_failed", "path", source.Path, "error", err)
			return nil, SourceError{Kind: source.Kind, Operation: "read_file", Err: err}
		}
		l.logger.InfoContext(ctx, "consumer_hazelcast_config_file_read_completed", "path", source.Path, "bytes", len(data))
		return data, nil
	case SourceS3:
		client, err := l.resolveAWSClient(source)
		if err != nil {
			return nil, err
		}
		output, err := client.GetS3(ctx, consumeraws.S3GetInput{Bucket: source.Bucket, Key: source.Key})
		if err != nil {
			return nil, SourceError{Kind: source.Kind, Operation: "get_s3_object", Err: err}
		}
		return output.Body, nil
	case SourceSecretsManager:
		client, err := l.resolveAWSClient(source)
		if err != nil {
			return nil, err
		}
		output, err := client.GetSecret(ctx, consumeraws.SecretGetInput{
			SecretID:     source.SecretID,
			VersionID:    source.VersionID,
			VersionStage: source.VersionStage,
		})
		if err != nil {
			return nil, SourceError{Kind: source.Kind, Operation: "get_secret", Err: err}
		}
		return output.Bytes(), nil
	default:
		return nil, InvalidConfigError{Field: "Source.Kind", Reason: "unsupported value"}
	}
}

func (l defaultSourceLoader) resolveAWSClient(source Source) (*consumeraws.Client, error) {
	if l.awsClient != nil {
		return l.awsClient, nil
	}
	client, err := consumeraws.New(consumeraws.Config{Region: source.AWSRegion})
	if err != nil {
		return nil, SourceError{Kind: source.Kind, Operation: "new_aws_consumer", Err: err}
	}
	l.awsClient = client
	return client, nil
}
