// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer tests the public Consumer service behavior.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

func TestREST_WithTokenAndJSONBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer abc" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer abc")
		}
		if got := r.Header.Get("X-App"); got != "orders-api" {
			t.Fatalf("X-App header = %q, want %q", got, "orders-api")
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page query = %q, want %q", got, "1")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	client, err := consumer.New(consumer.Config{},
		consumer.WithLogger(discardLogger()),
		consumer.WithTokenProvider(fakeTokenProvider{token: fakeToken("Bearer abc")}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	response, err := client.REST(http.MethodPost, server.URL).
		WithHeader("X-App", "orders-api").
		WithQueryParam("page", "1").
		WithBody(map[string]string{"id": "123"}).
		WithToken().
		Do(context.Background())
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var decoded struct {
		OK bool `json:"ok"`
	}
	if err := response.DecodeJSON(&decoded); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if !decoded.OK {
		t.Fatal("decoded OK = false, want true")
	}
}

func TestREST_WithTokenWithoutProviderReturnsTypedError(t *testing.T) {
	t.Parallel()

	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.REST(http.MethodGet, "https://example.com").WithToken().Do(context.Background())
	var tokenErr types.TokenRequiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("err = %T, want TokenRequiredError", err)
	}
}

func TestDynamoDBOperationsUseClientAndDecodeTargets(t *testing.T) {
	t.Parallel()

	db := &fakeDynamoDBClient{}
	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()), consumer.WithDynamoDBClient(db))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	item := record{PK: "CUSTOMER#1", SK: "PROFILE", Name: "Ana"}
	if err := client.PutDynamoDB(context.Background(), types.DynamoDBPutInput{TableName: "customers", Item: item}); err != nil {
		t.Fatalf("PutDynamoDB: %v", err)
	}
	if db.putInput == nil || aws.ToString(db.putInput.TableName) != "customers" {
		t.Fatalf("put table = %#v, want customers", db.putInput)
	}

	var updated record
	updateOutput, err := client.UpdateDynamoDB(context.Background(), types.DynamoDBUpdateInput{
		TableName:        "customers",
		Key:              map[string]string{"PK": "CUSTOMER#1", "SK": "PROFILE"},
		UpdateExpression: "SET #name = :name",
		ExpressionAttributeNames: map[string]string{
			"#name": "Name",
		},
		ExpressionAttributeValues: map[string]any{
			":name": "Ana",
		},
		ReturnValues: dynamodbtypes.ReturnValueAllNew,
		Target:       &updated,
	})
	if err != nil {
		t.Fatalf("UpdateDynamoDB: %v", err)
	}
	if len(updateOutput.Attributes) == 0 || updated.Name != "Ana" {
		t.Fatalf("update output = %#v, target = %#v", updateOutput, updated)
	}

	var got record
	output, err := client.GetDynamoDB(context.Background(), types.DynamoDBGetInput{
		TableName: "customers",
		Key:       map[string]string{"PK": "CUSTOMER#1", "SK": "PROFILE"},
		Target:    &got,
	})
	if err != nil {
		t.Fatalf("GetDynamoDB: %v", err)
	}
	if !output.Found || got.Name != "Ana" {
		t.Fatalf("get output = %#v, target = %#v", output, got)
	}

	var queried []record
	queryOutput, err := client.QueryDynamoDB(context.Background(), types.DynamoDBQueryInput{
		TableName:              "customers",
		KeyConditionExpression: "PK = :pk",
		ExpressionAttributeValues: map[string]any{
			":pk": "CUSTOMER#1",
		},
		Target: &queried,
	})
	if err != nil {
		t.Fatalf("QueryDynamoDB: %v", err)
	}
	if queryOutput.Count != 1 || len(queried) != 1 || queried[0].Name != "Ana" {
		t.Fatalf("query output = %#v, target = %#v", queryOutput, queried)
	}
}

func TestS3OperationsUseClient(t *testing.T) {
	t.Parallel()

	s3Client := &fakeS3Client{}
	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()), consumer.WithS3Client(s3Client))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	putOutput, err := client.PutS3(context.Background(), types.S3PutInput{
		Bucket:      "docs",
		Key:         "a.txt",
		Body:        "hello",
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("PutS3: %v", err)
	}
	if putOutput.ETag != `"etag"` {
		t.Fatalf("ETag = %q, want %q", putOutput.ETag, `"etag"`)
	}
	if got := readAllString(t, s3Client.putInput.Body); got != "hello" {
		t.Fatalf("put body = %q, want %q", got, "hello")
	}

	getOutput, err := client.GetS3(context.Background(), types.S3GetInput{Bucket: "docs", Key: "a.txt"})
	if err != nil {
		t.Fatalf("GetS3: %v", err)
	}
	if string(getOutput.Body) != "hello-from-s3" || getOutput.ContentType != "text/plain" {
		t.Fatalf("get output = %#v", getOutput)
	}
}

func TestSecretsManagerOperationsUseClientAndDecodeJSON(t *testing.T) {
	t.Parallel()

	secretsClient := &fakeSecretsManagerClient{}
	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()), consumer.WithSecretsManagerClient(secretsClient))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var secret databaseSecret
	getOutput, err := client.GetSecretJSON(context.Background(), types.SecretGetInput{SecretID: "orders/database"}, &secret)
	if err != nil {
		t.Fatalf("GetSecretJSON: %v", err)
	}
	if getOutput.Name != "orders/database" || secret.Username != "orders" || secret.Password != "secret" {
		t.Fatalf("get output = %#v, target = %#v", getOutput, secret)
	}

	putOutput, err := client.PutSecret(context.Background(), types.SecretPutInput{
		SecretID:           "orders/database",
		SecretString:       `{"username":"orders","password":"new"}`,
		ClientRequestToken: "token-1",
		VersionStages:      []string{"AWSCURRENT"},
	})
	if err != nil {
		t.Fatalf("PutSecret: %v", err)
	}
	if putOutput.VersionID != "version-put" || aws.ToString(secretsClient.putInput.SecretString) == "" {
		t.Fatalf("put output = %#v, input = %#v", putOutput, secretsClient.putInput)
	}

	updateOutput, err := client.UpdateSecret(context.Background(), types.SecretUpdateInput{
		SecretID:     "orders/database",
		Description:  "database credentials",
		SecretString: `{"username":"orders","password":"updated"}`,
	})
	if err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}
	if updateOutput.VersionID != "version-update" || aws.ToString(secretsClient.updateInput.Description) != "database credentials" {
		t.Fatalf("update output = %#v, input = %#v", updateOutput, secretsClient.updateInput)
	}
}

func TestSecretsManagerOperationReturnsTypedError(t *testing.T) {
	t.Parallel()

	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()), consumer.WithSecretsManagerClient(&fakeSecretsManagerClient{err: errors.New("boom")}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.GetSecret(context.Background(), types.SecretGetInput{SecretID: "orders/database"})
	var secretsErr types.SecretsManagerError
	if !errors.As(err, &secretsErr) {
		t.Fatalf("err = %T, want SecretsManagerError", err)
	}
}

func TestSQSOperationsUseClient(t *testing.T) {
	t.Parallel()

	sqsClient := &fakeSQSClient{}
	client, err := consumer.New(consumer.Config{}, consumer.WithLogger(discardLogger()), consumer.WithSQSClient(sqsClient))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sendOutput, err := client.SendSQS(context.Background(), types.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     "payload",
		MessageAttributes: map[string]string{
			"eventType": "OrderCreated",
		},
	})
	if err != nil {
		t.Fatalf("SendSQS: %v", err)
	}
	if sendOutput.MessageID != "msg-1" {
		t.Fatalf("MessageID = %q, want %q", sendOutput.MessageID, "msg-1")
	}

	receiveOutput, err := client.ReceiveSQS(context.Background(), types.SQSReceiveInput{
		QueueURL:              "https://sqs.us-east-1.amazonaws.com/123/orders",
		MaxNumberOfMessages:   1,
		WaitTimeSeconds:       5,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		t.Fatalf("ReceiveSQS: %v", err)
	}
	if len(receiveOutput.Messages) != 1 || receiveOutput.Messages[0].MessageAttributes["eventType"] != "OrderCreated" {
		t.Fatalf("receive output = %#v", receiveOutput)
	}

	if err := client.DeleteSQS(context.Background(), types.SQSDeleteInput{
		QueueURL:      "https://sqs.us-east-1.amazonaws.com/123/orders",
		ReceiptHandle: "receipt-1",
	}); err != nil {
		t.Fatalf("DeleteSQS: %v", err)
	}
	if aws.ToString(sqsClient.deleteInput.ReceiptHandle) != "receipt-1" {
		t.Fatalf("receipt handle = %q, want receipt-1", aws.ToString(sqsClient.deleteInput.ReceiptHandle))
	}
}

type fakeToken string

func (t fakeToken) ToString() string {
	return string(t)
}

type fakeTokenProvider struct {
	token fakeToken
}

func (p fakeTokenProvider) Token() types.AuthorizationToken {
	return p.token
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
	return &s3.PutObjectOutput{ETag: aws.String(`"etag"`)}, nil
}

func (c *fakeS3Client) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(strings.NewReader("hello-from-s3")),
		ContentType: aws.String("text/plain"),
		Metadata:    map[string]string{"source": "test"},
	}, nil
}

type fakeSecretsManagerClient struct {
	err         error
	putInput    *secretsmanager.PutSecretValueInput
	updateInput *secretsmanager.UpdateSecretInput
}

func (c *fakeSecretsManagerClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &secretsmanager.GetSecretValueOutput{
		ARN:           aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:          aws.String("orders/database"),
		VersionId:     aws.String("version-get"),
		VersionStages: []string{"AWSCURRENT"},
		SecretString:  aws.String(`{"username":"orders","password":"secret"}`),
	}, nil
}

func (c *fakeSecretsManagerClient) PutSecretValue(_ context.Context, input *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.putInput = input
	return &secretsmanager.PutSecretValueOutput{
		ARN:           aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:          aws.String("orders/database"),
		VersionId:     aws.String("version-put"),
		VersionStages: []string{"AWSCURRENT"},
	}, nil
}

func (c *fakeSecretsManagerClient) UpdateSecret(_ context.Context, input *secretsmanager.UpdateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.updateInput = input
	return &secretsmanager.UpdateSecretOutput{
		ARN:       aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:      aws.String("orders/database"),
		VersionId: aws.String("version-update"),
	}, nil
}

type fakeSQSClient struct {
	deleteInput *sqs.DeleteMessageInput
}

func (c *fakeSQSClient) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{MessageId: aws.String("msg-1")}, nil
}

func (c *fakeSQSClient) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{
		Messages: []sqstypes.Message{
			{
				MessageId:     aws.String("msg-1"),
				ReceiptHandle: aws.String("receipt-1"),
				Body:          aws.String("payload"),
				MessageAttributes: map[string]sqstypes.MessageAttributeValue{
					"eventType": {DataType: aws.String("String"), StringValue: aws.String("OrderCreated")},
				},
			},
		},
	}, nil
}

func (c *fakeSQSClient) DeleteMessage(_ context.Context, input *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	c.deleteInput = input
	return &sqs.DeleteMessageOutput{}, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func readAllString(t *testing.T, reader io.Reader) string {
	t.Helper()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(body)
}
