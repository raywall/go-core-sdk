package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/raywall/go-core-sdk/services/consumer"
	consumertypes "github.com/raywall/go-core-sdk/services/consumer/types"
)

func main() {
	ctx := context.Background()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer api.Close()

	client, err := consumer.New(consumer.Config{},
		consumer.WithTokenProvider(staticTokenProvider{}),
		consumer.WithDynamoDBClient(&fakeDynamoDBClient{}),
		consumer.WithS3Client(&fakeS3Client{}),
		consumer.WithSQSClient(&fakeSQSClient{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	restResponse, err := client.REST(http.MethodPost, api.URL).
		WithHeader("X-App", "orders-api").
		WithBody(map[string]any{"customerId": "CUSTOMER#1"}).
		WithToken().
		Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var order map[string]string
	if err := restResponse.DecodeJSON(&order); err != nil {
		log.Fatal(err)
	}

	if err := client.PutDynamoDB(ctx, consumertypes.DynamoDBPutInput{
		TableName: "orders",
		Item:      order,
	}); err != nil {
		log.Fatal(err)
	}

	var stored orderRecord
	getOutput, err := client.GetDynamoDB(ctx, consumertypes.DynamoDBGetInput{
		TableName: "orders",
		Key:       map[string]string{"id": "ORDER#1"},
		Target:    &stored,
	})
	if err != nil {
		log.Fatal(err)
	}

	s3Output, err := client.PutS3(ctx, consumertypes.S3PutInput{
		Bucket:      "orders-files",
		Key:         "ORDER#1.json",
		Body:        restResponse.Body,
		ContentType: "application/json",
	})
	if err != nil {
		log.Fatal(err)
	}

	sqsOutput, err := client.SendSQS(ctx, consumertypes.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     string(restResponse.Body),
		MessageAttributes: map[string]string{
			"eventType": "OrderCreated",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	receiveOutput, err := client.ReceiveSQS(ctx, consumertypes.SQSReceiveInput{
		QueueURL:              "https://sqs.us-east-1.amazonaws.com/123/orders",
		MaxNumberOfMessages:   1,
		WaitTimeSeconds:       1,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("restStatus=%d order=%s found=%t storedStatus=%s s3ETag=%s sqsMessage=%s received=%d\n",
		restResponse.StatusCode,
		order["id"],
		getOutput.Found,
		stored.Status,
		s3Output.ETag,
		sqsOutput.MessageID,
		len(receiveOutput.Messages),
	)
}

type staticTokenProvider struct{}

func (staticTokenProvider) Token() consumertypes.AuthorizationToken {
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
