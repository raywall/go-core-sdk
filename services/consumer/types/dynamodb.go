// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public DynamoDB request and response contracts.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

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
