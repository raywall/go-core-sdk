// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws tests the public AWS consumer behavior.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

func TestClient_DynamoDBOperationsUseConfiguredClient(t *testing.T) {
	t.Parallel()

	db := &fakeDynamoDBClient{}
	client, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(discardLogger()), consumeraws.WithDynamoDBClient(db))
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	item := record{PK: "CUSTOMER#1", SK: "PROFILE", Name: "Ana"}
	if err := client.PutDynamoDB(context.Background(), consumeraws.DynamoDBPutInput{TableName: "customers", Item: item}); err != nil {
		t.Fatalf("PutDynamoDB() error = %v", err)
	}
	if db.putInput == nil || awssdk.ToString(db.putInput.TableName) != "customers" {
		t.Fatalf("put table = %#v, want customers", db.putInput)
	}

	var got record
	output, err := client.GetDynamoDB(context.Background(), consumeraws.DynamoDBGetInput{
		TableName: "customers",
		Key:       map[string]string{"PK": "CUSTOMER#1", "SK": "PROFILE"},
		Target:    &got,
	})
	if err != nil {
		t.Fatalf("GetDynamoDB() error = %v", err)
	}
	if !output.Found || got.Name != "Ana" {
		t.Fatalf("get output = %#v, target = %#v", output, got)
	}
}

func TestClient_S3OperationsUseConfiguredClient(t *testing.T) {
	t.Parallel()

	s3Client := &fakeS3Client{}
	client, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(discardLogger()), consumeraws.WithS3Client(s3Client))
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	putOutput, err := client.PutS3(context.Background(), consumeraws.S3PutInput{
		Bucket: "docs",
		Key:    "a.txt",
		Body:   "hello",
	})
	if err != nil {
		t.Fatalf("PutS3() error = %v", err)
	}
	if putOutput.ETag != `"etag"` {
		t.Fatalf("ETag = %q, want %q", putOutput.ETag, `"etag"`)
	}
	if got := readAllString(t, s3Client.putInput.Body); got != "hello" {
		t.Fatalf("put body = %q, want %q", got, "hello")
	}
}

func TestClient_S3UsesPathStyleWithConfiguredEndpoint(t *testing.T) {
	t.Parallel()

	type capturedRequest struct {
		host string
		path string
	}
	requests := make(chan capturedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- capturedRequest{host: r.Host, path: r.URL.Path}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := consumeraws.New(
		consumeraws.Config{
			Region:         "us-east-1",
			EndpointURL:    server.URL,
			S3UsePathStyle: true,
		},
		consumeraws.WithLogger(discardLogger()),
		consumeraws.WithConfig(awssdk.Config{
			Region: "us-east-1",
			Credentials: awssdk.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
				"access-key",
				"secret-key",
				"",
			)),
		}),
	)
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	_, err = client.PutS3(context.Background(), consumeraws.S3PutInput{
		Bucket: "docs",
		Key:    "a.txt",
		Body:   "hello",
	})
	if err != nil {
		t.Fatalf("PutS3() error = %v", err)
	}

	got := <-requests
	wantHost := strings.TrimPrefix(server.URL, "http://")
	if got.host != wantHost {
		t.Fatalf("request host = %q, want %q", got.host, wantHost)
	}
	if got.path != "/docs/a.txt" {
		t.Fatalf("request path = %q, want /docs/a.txt", got.path)
	}
}

func TestClient_SecretsManagerOperationsUseConfiguredClientAndTypedErrors(t *testing.T) {
	t.Parallel()

	secretsClient := &fakeSecretsManagerClient{}
	client, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(discardLogger()), consumeraws.WithSecretsManagerClient(secretsClient))
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	var secret databaseSecret
	output, err := client.GetSecretJSON(context.Background(), consumeraws.SecretGetInput{SecretID: "orders/database"}, &secret)
	if err != nil {
		t.Fatalf("GetSecretJSON() error = %v", err)
	}
	if output.Name != "orders/database" || secret.Username != "orders" {
		t.Fatalf("output = %#v, secret = %#v", output, secret)
	}

	failing, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(discardLogger()), consumeraws.WithSecretsManagerClient(&fakeSecretsManagerClient{err: errors.New("boom")}))
	if err != nil {
		t.Fatalf("aws.New() failing error = %v", err)
	}
	_, err = failing.GetSecret(context.Background(), consumeraws.SecretGetInput{SecretID: "orders/database"})
	var secretsErr consumeraws.SecretsManagerError
	if !errors.As(err, &secretsErr) {
		t.Fatalf("err = %T, want SecretsManagerError", err)
	}
}

func TestClient_SQSOperationsUseConfiguredClient(t *testing.T) {
	t.Parallel()

	sqsClient := &fakeSQSClient{}
	client, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(discardLogger()), consumeraws.WithSQSClient(sqsClient))
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	sendOutput, err := client.SendSQS(context.Background(), consumeraws.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     "payload",
		MessageAttributes: map[string]string{
			"eventType": "OrderCreated",
		},
	})
	if err != nil {
		t.Fatalf("SendSQS() error = %v", err)
	}
	if sendOutput.MessageID != "msg-1" {
		t.Fatalf("MessageID = %q, want msg-1", sendOutput.MessageID)
	}

	receiveOutput, err := client.ReceiveSQS(context.Background(), consumeraws.SQSReceiveInput{QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders"})
	if err != nil {
		t.Fatalf("ReceiveSQS() error = %v", err)
	}
	if len(receiveOutput.Messages) != 1 || receiveOutput.Messages[0].MessageAttributes["eventType"] != "OrderCreated" {
		t.Fatalf("receive output = %#v", receiveOutput)
	}
}

type record struct {
	PK   string `dynamodbav:"PK"`
	SK   string `dynamodbav:"SK"`
	Name string `dynamodbav:"Name"`
}

type databaseSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type fakeDynamoDBClient struct {
	putInput *dynamodb.PutItemInput
}

func (c *fakeDynamoDBClient) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	c.putInput = input
	return &dynamodb.PutItemOutput{}, nil
}

func (c *fakeDynamoDBClient) UpdateItem(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	item, err := attributevalue.MarshalMap(record{PK: "CUSTOMER#1", SK: "PROFILE", Name: "Ana"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.UpdateItemOutput{Attributes: item}, nil
}

func (c *fakeDynamoDBClient) GetItem(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	item, err := attributevalue.MarshalMap(record{PK: "CUSTOMER#1", SK: "PROFILE", Name: "Ana"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.GetItemOutput{Item: item}, nil
}

func (c *fakeDynamoDBClient) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	item, err := attributevalue.MarshalMap(record{PK: "CUSTOMER#1", SK: "PROFILE", Name: "Ana"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.QueryOutput{
		Count: 1,
		Items: []map[string]dynamodbtypes.AttributeValue{item},
	}, nil
}

type fakeS3Client struct {
	putInput *s3.PutObjectInput
}

func (c *fakeS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	c.putInput = input
	return &s3.PutObjectOutput{ETag: awssdk.String(`"etag"`)}, nil
}

func (c *fakeS3Client) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(strings.NewReader("hello-from-s3")),
		ContentType: awssdk.String("text/plain"),
	}, nil
}

type fakeSecretsManagerClient struct {
	err error
}

func (c *fakeSecretsManagerClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &secretsmanager.GetSecretValueOutput{
		Name:         awssdk.String("orders/database"),
		VersionId:    awssdk.String("version-get"),
		SecretString: awssdk.String(`{"username":"orders","password":"secret"}`),
	}, nil
}

func (c *fakeSecretsManagerClient) PutSecretValue(_ context.Context, _ *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (c *fakeSecretsManagerClient) UpdateSecret(_ context.Context, _ *secretsmanager.UpdateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &secretsmanager.UpdateSecretOutput{}, nil
}

type fakeSQSClient struct{}

func (c *fakeSQSClient) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{MessageId: awssdk.String("msg-1")}, nil
}

func (c *fakeSQSClient) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{
		Messages: []sqstypes.Message{
			{
				MessageId:     awssdk.String("msg-1"),
				ReceiptHandle: awssdk.String("receipt-1"),
				Body:          awssdk.String("payload"),
				MessageAttributes: map[string]sqstypes.MessageAttributeValue{
					"eventType": {DataType: awssdk.String("String"), StringValue: awssdk.String("OrderCreated")},
				},
			},
		},
	}, nil
}

func (c *fakeSQSClient) DeleteMessage(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func readAllString(t *testing.T, reader io.Reader) string {
	t.Helper()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(body)
}
