// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws implements public AWS request, response and client contracts.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DynamoDBClient defines the DynamoDB operations used by Client.
type DynamoDBClient interface {
	// PutItem creates or replaces an item.
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	// UpdateItem updates an existing item and may return updated attributes.
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	// GetItem retrieves a single item by key.
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	// Query retrieves items matching a key condition expression.
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// S3Client defines the S3 operations used by Client.
type S3Client interface {
	// PutObject stores an object in a bucket.
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	// GetObject retrieves an object from a bucket.
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// SecretsManagerClient defines the Secrets Manager operations used by Client.
type SecretsManagerClient interface {
	// GetSecretValue retrieves the current or requested value of a secret.
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	// PutSecretValue adds a new version value to an existing secret.
	PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	// UpdateSecret updates secret metadata and optionally creates a new secret value.
	UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
}

// SQSClient defines the SQS operations used by Client.
type SQSClient interface {
	// SendMessage sends a message to a queue.
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	// ReceiveMessage reads messages from a queue.
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	// DeleteMessage deletes a message from a queue by receipt handle.
	DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// DynamoDBPutInput describes an item insertion or replacement.
type DynamoDBPutInput struct {
	// TableName is the target DynamoDB table.
	TableName string
	// Item is a struct, map or raw DynamoDB attribute map.
	Item any
	// ConditionExpression optionally guards the put operation.
	ConditionExpression string
	// ExpressionAttributeNames aliases attribute names used by expressions.
	ExpressionAttributeNames map[string]string
	// ExpressionAttributeValues provides expression values as Go values.
	ExpressionAttributeValues map[string]any
}

// DynamoDBUpdateInput describes an item update operation.
type DynamoDBUpdateInput struct {
	// TableName is the target DynamoDB table.
	TableName string
	// Key is a struct, map or raw DynamoDB attribute map containing the item key.
	Key any
	// UpdateExpression defines the DynamoDB update expression.
	UpdateExpression string
	// ConditionExpression optionally guards the update operation.
	ConditionExpression string
	// ExpressionAttributeNames aliases attribute names used by expressions.
	ExpressionAttributeNames map[string]string
	// ExpressionAttributeValues provides expression values as Go values.
	ExpressionAttributeValues map[string]any
	// ReturnValues controls which attributes DynamoDB returns after the update.
	ReturnValues dynamodbtypes.ReturnValue
	// Target receives returned attributes when provided.
	Target any
}

// DynamoDBUpdateOutput contains attributes returned by UpdateItem.
type DynamoDBUpdateOutput struct {
	// Attributes contains raw DynamoDB attributes returned by the service.
	Attributes map[string]dynamodbtypes.AttributeValue
}

// DynamoDBGetInput describes a get operation by key.
type DynamoDBGetInput struct {
	// TableName is the target DynamoDB table.
	TableName string
	// Key is a struct, map or raw DynamoDB attribute map containing the item key.
	Key any
	// ConsistentRead controls DynamoDB strongly consistent reads.
	ConsistentRead bool
	// ProjectionExpression restricts attributes returned by DynamoDB.
	ProjectionExpression string
	// ExpressionAttributeNames aliases attribute names used by expressions.
	ExpressionAttributeNames map[string]string
	// Target receives the decoded item when provided.
	Target any
}

// DynamoDBGetOutput contains the item returned by GetItem.
type DynamoDBGetOutput struct {
	// Found indicates whether DynamoDB returned an item.
	Found bool
	// Item contains the raw DynamoDB item when found.
	Item map[string]dynamodbtypes.AttributeValue
}

// DynamoDBQueryInput describes a query operation.
type DynamoDBQueryInput struct {
	// TableName is the target DynamoDB table.
	TableName string
	// KeyConditionExpression defines the required DynamoDB key condition.
	KeyConditionExpression string
	// FilterExpression optionally filters items after the key condition.
	FilterExpression string
	// ProjectionExpression restricts attributes returned by DynamoDB.
	ProjectionExpression string
	// ExpressionAttributeNames aliases attribute names used by expressions.
	ExpressionAttributeNames map[string]string
	// ExpressionAttributeValues provides expression values as Go values.
	ExpressionAttributeValues map[string]any
	// IndexName optionally selects a secondary index.
	IndexName string
	// Limit optionally limits the number of items returned.
	Limit int32
	// ScanIndexForward controls sort key ordering when set.
	ScanIndexForward *bool
	// Target receives decoded items when provided. Pass a pointer to a slice.
	Target any
}

// DynamoDBQueryOutput contains items returned by Query.
type DynamoDBQueryOutput struct {
	// Count is the number of matching items returned by DynamoDB.
	Count int32
	// Items contains the raw DynamoDB items returned by the service.
	Items []map[string]dynamodbtypes.AttributeValue
}

// S3PutInput describes an object upload.
type S3PutInput struct {
	// Bucket is the target S3 bucket.
	Bucket string
	// Key is the target object key.
	Key string
	// Body is the object payload. Supported values include []byte, string and io.Reader.
	Body any
	// ContentType optionally sets the object content type.
	ContentType string
	// Metadata contains custom object metadata.
	Metadata map[string]string
}

// S3PutOutput contains metadata returned by S3 after an upload.
type S3PutOutput struct {
	// ETag is the entity tag returned by S3 when available.
	ETag string
}

// S3GetInput describes an object download.
type S3GetInput struct {
	// Bucket is the source S3 bucket.
	Bucket string
	// Key is the source object key.
	Key string
}

// S3GetOutput contains a downloaded S3 object.
type S3GetOutput struct {
	// Body contains the full object payload.
	Body []byte
	// ContentType is the object content type returned by S3.
	ContentType string
	// Metadata contains custom object metadata returned by S3.
	Metadata map[string]string
}

// SecretGetInput describes a secret read operation.
type SecretGetInput struct {
	// SecretID is the secret name or ARN.
	SecretID string
	// VersionID optionally selects a specific secret version.
	VersionID string
	// VersionStage optionally selects a version stage, such as AWSCURRENT.
	VersionStage string
}

// SecretGetOutput contains a secret value returned by Secrets Manager.
type SecretGetOutput struct {
	// ARN is the secret ARN returned by Secrets Manager.
	ARN string
	// Name is the secret name returned by Secrets Manager.
	Name string
	// VersionID is the version identifier returned by Secrets Manager.
	VersionID string
	// VersionStages contains labels attached to the returned version.
	VersionStages []string
	// SecretString contains the secret string when the secret is textual.
	SecretString string
	// SecretBinary contains the secret binary value when the secret is binary.
	SecretBinary []byte
}

// Bytes returns the secret value as bytes.
//
// Bytes returns SecretString bytes when SecretString is present. Otherwise it
// returns a copy of SecretBinary.
func (o SecretGetOutput) Bytes() []byte {
	if o.SecretString != "" {
		return []byte(o.SecretString)
	}
	return append([]byte(nil), o.SecretBinary...)
}

// SecretPutInput describes adding a new value version to an existing secret.
type SecretPutInput struct {
	// SecretID is the secret name or ARN.
	SecretID string
	// SecretString is the textual secret value. Do not set with SecretBinary.
	SecretString string
	// SecretBinary is the binary secret value. Do not set with SecretString.
	SecretBinary []byte
	// ClientRequestToken optionally supplies an idempotency token.
	ClientRequestToken string
	// VersionStages optionally labels the new secret version.
	VersionStages []string
}

// SecretPutOutput contains metadata returned after adding a secret value.
type SecretPutOutput struct {
	// ARN is the secret ARN returned by Secrets Manager.
	ARN string
	// Name is the secret name returned by Secrets Manager.
	Name string
	// VersionID is the new version identifier.
	VersionID string
	// VersionStages contains labels attached to the new version.
	VersionStages []string
}

// SecretUpdateInput describes updating an existing secret.
type SecretUpdateInput struct {
	// SecretID is the secret name or ARN.
	SecretID string
	// Description optionally replaces the secret description.
	Description string
	// KMSKeyID optionally replaces the KMS key used for encryption.
	KMSKeyID string
	// SecretString is the textual secret value. Do not set with SecretBinary.
	SecretString string
	// SecretBinary is the binary secret value. Do not set with SecretString.
	SecretBinary []byte
	// ClientRequestToken optionally supplies an idempotency token.
	ClientRequestToken string
}

// SecretUpdateOutput contains metadata returned after updating a secret.
type SecretUpdateOutput struct {
	// ARN is the secret ARN returned by Secrets Manager.
	ARN string
	// Name is the secret name returned by Secrets Manager.
	Name string
	// VersionID is the new version identifier when a new value was created.
	VersionID string
}

// SQSSendInput describes a message send operation.
type SQSSendInput struct {
	// QueueURL is the target SQS queue URL.
	QueueURL string
	// Body is the message body.
	Body string
	// DelaySeconds optionally delays message visibility.
	DelaySeconds int32
	// MessageAttributes contains string message attributes.
	MessageAttributes map[string]string
}

// SQSSendOutput contains metadata returned after sending a message.
type SQSSendOutput struct {
	// MessageID is the SQS message identifier.
	MessageID string
}

// SQSReceiveInput describes a message receive operation.
type SQSReceiveInput struct {
	// QueueURL is the source SQS queue URL.
	QueueURL string
	// MaxNumberOfMessages limits how many messages are returned.
	MaxNumberOfMessages int32
	// WaitTimeSeconds enables long polling when greater than zero.
	WaitTimeSeconds int32
	// VisibilityTimeout optionally overrides message visibility timeout.
	VisibilityTimeout int32
	// AttributeNames lists system attributes to retrieve.
	AttributeNames []string
	// MessageAttributeNames lists custom message attributes to retrieve.
	MessageAttributeNames []string
}

// SQSReceiveOutput contains messages returned by SQS.
type SQSReceiveOutput struct {
	// Messages contains the received messages.
	Messages []SQSMessage
}

// SQSMessage contains the essential values of an SQS message.
type SQSMessage struct {
	// MessageID is the SQS message identifier.
	MessageID string
	// ReceiptHandle is required to delete the message after processing.
	ReceiptHandle string
	// Body is the message body.
	Body string
	// Attributes contains system attributes returned by SQS.
	Attributes map[string]string
	// MessageAttributes contains string custom message attributes returned by SQS.
	MessageAttributes map[string]string
}

// SQSDeleteInput describes a message delete operation.
type SQSDeleteInput struct {
	// QueueURL is the source SQS queue URL.
	QueueURL string
	// ReceiptHandle identifies the message to delete.
	ReceiptHandle string
}
