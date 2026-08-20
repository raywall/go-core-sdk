package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/raywall/go-core-sdk/services/consumer"
)

func TestRunConsumesRESTAndAWSLikeAdapters(t *testing.T) {
	t.Parallel()

	api := newOrdersAPI()
	defer api.Close()

	client, err := consumer.New(consumer.Config{},
		consumer.WithTokenProvider(staticTokenProvider{}),
		consumer.WithDynamoDBClient(&fakeDynamoDBClient{}),
		consumer.WithS3Client(&fakeS3Client{}),
		consumer.WithSecretsManagerClient(&fakeSecretsManagerClient{}),
		consumer.WithSQSClient(&fakeSQSClient{}),
	)
	if err != nil {
		t.Fatalf("consumer.New() error = %v", err)
	}

	var out bytes.Buffer
	if err := run(context.Background(), OrdersUseCase{Client: client, APIURL: api.URL, Output: &out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"restStatus=200", "order=ORDER#1", "sqsMessage=message-1", "secretUsername=orders"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
