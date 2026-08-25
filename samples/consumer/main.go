package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	consumerrest "github.com/raywall/go-core-sdk/services/consumer/rest"
)

func main() {
	ctx := context.Background()
	api := newOrdersAPI()
	defer api.Close()

	restClient, err := consumerrest.New(consumerrest.Config{},
		consumerrest.WithTokenProvider(staticTokenProvider{}),
	)
	if err != nil {
		log.Fatal(err)
	}
	awsClient, err := consumeraws.New(consumeraws.Config{},
		consumeraws.WithDynamoDBClient(&fakeDynamoDBClient{}),
		consumeraws.WithS3Client(&fakeS3Client{}),
		consumeraws.WithSecretsManagerClient(&fakeSecretsManagerClient{}),
		consumeraws.WithSQSClient(&fakeSQSClient{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx, OrdersUseCase{
		REST:   restClient,
		AWS:    awsClient,
		APIURL: api.URL,
		Output: os.Stdout,
	}); err != nil {
		log.Fatal(err)
	}
}

// OrdersUseCase demonstrates outbound REST and AWS-like consumer adapters.
type OrdersUseCase struct {
	REST   *consumerrest.Client
	AWS    *consumeraws.Client
	APIURL string
	Output io.Writer
}

func run(ctx context.Context, useCase OrdersUseCase) error {
	return useCase.Execute(ctx)
}

func (u OrdersUseCase) Execute(ctx context.Context) error {
	restResponse, err := u.REST.REST(http.MethodPost, u.APIURL).
		WithHeader("X-App", "orders-api").
		WithBody(map[string]any{"customerId": "CUSTOMER#1"}).
		WithToken().
		Do(ctx)
	if err != nil {
		return err
	}

	var order map[string]string
	if err := restResponse.DecodeJSON(&order); err != nil {
		return err
	}

	if err := u.AWS.PutDynamoDB(ctx, consumeraws.DynamoDBPutInput{
		TableName: "orders",
		Item:      order,
	}); err != nil {
		return err
	}

	var stored orderRecord
	getOutput, err := u.AWS.GetDynamoDB(ctx, consumeraws.DynamoDBGetInput{
		TableName: "orders",
		Key:       map[string]string{"id": "ORDER#1"},
		Target:    &stored,
	})
	if err != nil {
		return err
	}

	s3Output, err := u.AWS.PutS3(ctx, consumeraws.S3PutInput{
		Bucket:      "orders-files",
		Key:         "ORDER#1.json",
		Body:        restResponse.Body,
		ContentType: "application/json",
	})
	if err != nil {
		return err
	}

	sqsOutput, err := u.AWS.SendSQS(ctx, consumeraws.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     string(restResponse.Body),
		MessageAttributes: map[string]string{
			"eventType": "OrderCreated",
		},
	})
	if err != nil {
		return err
	}

	receiveOutput, err := u.AWS.ReceiveSQS(ctx, consumeraws.SQSReceiveInput{
		QueueURL:              "https://sqs.us-east-1.amazonaws.com/123/orders",
		MaxNumberOfMessages:   1,
		WaitTimeSeconds:       1,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return err
	}

	var database databaseSecret
	if _, err := u.AWS.GetSecretJSON(ctx, consumeraws.SecretGetInput{SecretID: "orders/database"}, &database); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(u.Output, "restStatus=%d order=%s found=%t storedStatus=%s s3ETag=%s sqsMessage=%s received=%d\n",
		restResponse.StatusCode,
		order["id"],
		getOutput.Found,
		stored.Status,
		s3Output.ETag,
		sqsOutput.MessageID,
		len(receiveOutput.Messages),
	); err != nil {
		return err
	}
	_, err = fmt.Fprintf(u.Output, "secretUsername=%s\n", database.Username)
	return err
}

func newOrdersAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sample-token" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "ORDER#1",
			"status": "CREATED",
		})
	}))
}

type staticTokenProvider struct{}

func (staticTokenProvider) Token() consumerrest.AuthorizationToken {
	return staticToken{}
}

type staticToken struct{}

func (staticToken) ToString() string {
	return "Bearer sample-token"
}

type orderRecord struct {
	ID     string `dynamodbav:"id"`
	Status string `dynamodbav:"status"`
}

type databaseSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type fakeDynamoDBClient struct{}

func (c *fakeDynamoDBClient) PutItem(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return &dynamodb.PutItemOutput{}, nil
}

func (c *fakeDynamoDBClient) UpdateItem(_ context.Context, _ *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	item, err := attributevalue.MarshalMap(orderRecord{ID: "ORDER#1", Status: "UPDATED"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.UpdateItemOutput{Attributes: item}, nil
}

func (c *fakeDynamoDBClient) GetItem(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	item, err := attributevalue.MarshalMap(orderRecord{ID: "ORDER#1", Status: "CREATED"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.GetItemOutput{Item: item}, nil
}

func (c *fakeDynamoDBClient) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	item, err := attributevalue.MarshalMap(orderRecord{ID: "ORDER#1", Status: "CREATED"})
	if err != nil {
		return nil, err
	}
	return &dynamodb.QueryOutput{
		Count: 1,
		Items: []map[string]dynamodbtypes.AttributeValue{item},
	}, nil
}

type fakeS3Client struct{}

func (c *fakeS3Client) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{ETag: aws.String(`"sample-etag"`)}, nil
}

func (c *fakeS3Client) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(strings.NewReader(`{"id":"ORDER#1","status":"CREATED"}`)),
		ContentType: aws.String("application/json"),
	}, nil
}

type fakeSecretsManagerClient struct{}

func (c *fakeSecretsManagerClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		ARN:           aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:          aws.String("orders/database"),
		VersionId:     aws.String("version-1"),
		VersionStages: []string{"AWSCURRENT"},
		SecretString:  aws.String(`{"username":"orders","password":"secret"}`),
	}, nil
}

func (c *fakeSecretsManagerClient) PutSecretValue(_ context.Context, _ *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	return &secretsmanager.PutSecretValueOutput{
		ARN:       aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:      aws.String("orders/database"),
		VersionId: aws.String("version-2"),
	}, nil
}

func (c *fakeSecretsManagerClient) UpdateSecret(_ context.Context, _ *secretsmanager.UpdateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return &secretsmanager.UpdateSecretOutput{
		ARN:       aws.String("arn:aws:secretsmanager:us-east-1:123:secret:orders/database"),
		Name:      aws.String("orders/database"),
		VersionId: aws.String("version-3"),
	}, nil
}

type fakeSQSClient struct{}

func (c *fakeSQSClient) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{MessageId: aws.String("message-1")}, nil
}

func (c *fakeSQSClient) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{
		Messages: []sqstypes.Message{
			{
				MessageId:     aws.String("message-1"),
				ReceiptHandle: aws.String("receipt-1"),
				Body:          aws.String(`{"id":"ORDER#1","status":"CREATED"}`),
				MessageAttributes: map[string]sqstypes.MessageAttributeValue{
					"eventType": {DataType: aws.String("String"), StringValue: aws.String("OrderCreated")},
				},
			},
		},
	}, nil
}

func (c *fakeSQSClient) DeleteMessage(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}
