package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	consumerrest "github.com/raywall/go-core-sdk/services/consumer/rest"
)

func TestRunConsumesRESTAndAWSLikeAdapters(t *testing.T) {
	t.Parallel()

	api := newOrdersAPI()
	defer api.Close()

	restClient, err := consumerrest.New(consumerrest.Config{},
		consumerrest.WithTokenProvider(staticTokenProvider{}),
	)
	if err != nil {
		t.Fatalf("rest.New() error = %v", err)
	}
	awsClient, err := consumeraws.New(consumeraws.Config{},
		consumeraws.WithDynamoDBClient(&fakeDynamoDBClient{}),
		consumeraws.WithS3Client(&fakeS3Client{}),
		consumeraws.WithSecretsManagerClient(&fakeSecretsManagerClient{}),
		consumeraws.WithSQSClient(&fakeSQSClient{}),
	)
	if err != nil {
		t.Fatalf("aws.New() error = %v", err)
	}

	var out bytes.Buffer
	if err := run(context.Background(), OrdersUseCase{REST: restClient, AWS: awsClient, APIURL: api.URL, Output: &out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"restStatus=200", "order=ORDER#1", "sqsMessage=message-1", "secretUsername=orders"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
