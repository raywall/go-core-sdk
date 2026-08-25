// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast tests distributed configuration source loading.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	"github.com/raywall/go-core-sdk/services/consumer/hazelcast"
)

func TestNew_LoadsS3Source(t *testing.T) {
	t.Parallel()

	awsClient := newAWSClient(t, awsFakes{
		s3: fakeS3Client{body: `{"Cluster":{"Name":"s3-config"}}`},
	})
	client, err := hazelcast.New(context.Background(), hazelcast.Config{
		Source: hazelcast.Source{Kind: hazelcast.SourceS3, Bucket: "configs", Key: "hazelcast/client.json"},
	}, hazelcast.WithLogger(discardLogger()), hazelcast.WithAWSClient(awsClient), hazelcast.WithStarter(fakeStarter(t, nil)))
	if err != nil {
		t.Fatalf("hazelcast.New() error = %v", err)
	}
	if got := client.HazelcastConfig().Cluster.Name; got != "s3-config" {
		t.Fatalf("HazelcastConfig().Cluster.Name = %q, want s3-config", got)
	}
}

func TestNew_LoadsSecretsManagerSource(t *testing.T) {
	t.Parallel()

	awsClient := newAWSClient(t, awsFakes{
		secrets: fakeSecretsManagerClient{secret: `{"Cluster":{"Name":"secret-config"}}`},
	})
	client, err := hazelcast.New(context.Background(), hazelcast.Config{
		Source: hazelcast.Source{Kind: hazelcast.SourceSecretsManager, SecretID: "app/hazelcast"},
	}, hazelcast.WithLogger(discardLogger()), hazelcast.WithAWSClient(awsClient), hazelcast.WithStarter(fakeStarter(t, nil)))
	if err != nil {
		t.Fatalf("hazelcast.New() error = %v", err)
	}
	if got := client.HazelcastConfig().Cluster.Name; got != "secret-config" {
		t.Fatalf("HazelcastConfig().Cluster.Name = %q, want secret-config", got)
	}
}

type awsFakes struct {
	s3      consumeraws.S3Client
	secrets consumeraws.SecretsManagerClient
}

func newAWSClient(t *testing.T, fakes awsFakes) *consumeraws.Client {
	t.Helper()
	client, err := consumeraws.New(consumeraws.Config{},
		consumeraws.WithLogger(discardLogger()),
		consumeraws.WithDynamoDBClient(fakeDynamoDBClient{}),
		consumeraws.WithS3Client(fakes.s3),
		consumeraws.WithSecretsManagerClient(fakes.secrets),
		consumeraws.WithSQSClient(fakeSQSClient{}),
	)
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}
	return client
}

type fakeS3Client struct {
	body string
}

func (c fakeS3Client) PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (c fakeS3Client) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(c.body))}, nil
}

type fakeSecretsManagerClient struct {
	secret string
}

func (c fakeSecretsManagerClient) GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(c.secret)}, nil
}

func (c fakeSecretsManagerClient) PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (c fakeSecretsManagerClient) UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return &secretsmanager.UpdateSecretOutput{}, nil
}

type fakeDynamoDBClient struct{}

func (fakeDynamoDBClient) PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return &dynamodb.PutItemOutput{}, nil
}

func (fakeDynamoDBClient) UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return &dynamodb.UpdateItemOutput{}, nil
}

func (fakeDynamoDBClient) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{}, nil
}

func (fakeDynamoDBClient) Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{}, nil
}

type fakeSQSClient struct{}

func (fakeSQSClient) SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{}, nil
}

func (fakeSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{}, nil
}

func (fakeSQSClient) DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}
