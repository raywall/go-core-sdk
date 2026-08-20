// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements DynamoDB convenience operations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// PutDynamoDB stores an item in a DynamoDB table.
func (c *Consumer) PutDynamoDB(ctx context.Context, input types.DynamoDBPutInput) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.dynamoDBClient(ctx)
	if err != nil {
		return types.DynamoDBError{Operation: "load_config", Err: err}
	}
	item, err := marshalDynamoMap(input.Item)
	if err != nil {
		return types.DynamoDBError{Operation: "marshal_item", Err: err}
	}
	values, err := marshalDynamoValues(input.ExpressionAttributeValues)
	if err != nil {
		return types.DynamoDBError{Operation: "marshal_expression_values", Err: err}
	}

	request := &dynamodb.PutItemInput{
		TableName:                 aws.String(strings.TrimSpace(input.TableName)),
		Item:                      item,
		ConditionExpression:       optionalString(input.ConditionExpression),
		ExpressionAttributeNames:  input.ExpressionAttributeNames,
		ExpressionAttributeValues: values,
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_put_started", "table", strings.TrimSpace(input.TableName))
	if _, err := client.PutItem(ctx, request); err != nil {
		c.logger.ErrorContext(ctx, "consumer_dynamodb_put_failed", "table", strings.TrimSpace(input.TableName), "error", err)
		return types.DynamoDBError{Operation: "put_item", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_put_completed", "table", strings.TrimSpace(input.TableName))
	return nil
}

// UpdateDynamoDB updates an item in a DynamoDB table.
func (c *Consumer) UpdateDynamoDB(ctx context.Context, input types.DynamoDBUpdateInput) (types.DynamoDBUpdateOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.dynamoDBClient(ctx)
	if err != nil {
		return types.DynamoDBUpdateOutput{}, types.DynamoDBError{Operation: "load_config", Err: err}
	}
	key, err := marshalDynamoMap(input.Key)
	if err != nil {
		return types.DynamoDBUpdateOutput{}, types.DynamoDBError{Operation: "marshal_key", Err: err}
	}
	values, err := marshalDynamoValues(input.ExpressionAttributeValues)
	if err != nil {
		return types.DynamoDBUpdateOutput{}, types.DynamoDBError{Operation: "marshal_expression_values", Err: err}
	}

	request := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(strings.TrimSpace(input.TableName)),
		Key:                       key,
		UpdateExpression:          optionalString(input.UpdateExpression),
		ConditionExpression:       optionalString(input.ConditionExpression),
		ExpressionAttributeNames:  input.ExpressionAttributeNames,
		ExpressionAttributeValues: values,
		ReturnValues:              input.ReturnValues,
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_update_started", "table", strings.TrimSpace(input.TableName))
	response, err := client.UpdateItem(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_dynamodb_update_failed", "table", strings.TrimSpace(input.TableName), "error", err)
		return types.DynamoDBUpdateOutput{}, types.DynamoDBError{Operation: "update_item", Err: err}
	}
	if input.Target != nil && len(response.Attributes) > 0 {
		if err := attributevalue.UnmarshalMap(response.Attributes, input.Target); err != nil {
			return types.DynamoDBUpdateOutput{}, types.DecodeError{Operation: "dynamodb_update_attributes", Err: err}
		}
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_update_completed", "table", strings.TrimSpace(input.TableName))
	return types.DynamoDBUpdateOutput{Attributes: response.Attributes}, nil
}

// GetDynamoDB retrieves one item from a DynamoDB table by key.
func (c *Consumer) GetDynamoDB(ctx context.Context, input types.DynamoDBGetInput) (types.DynamoDBGetOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.dynamoDBClient(ctx)
	if err != nil {
		return types.DynamoDBGetOutput{}, types.DynamoDBError{Operation: "load_config", Err: err}
	}
	key, err := marshalDynamoMap(input.Key)
	if err != nil {
		return types.DynamoDBGetOutput{}, types.DynamoDBError{Operation: "marshal_key", Err: err}
	}

	request := &dynamodb.GetItemInput{
		TableName:                aws.String(strings.TrimSpace(input.TableName)),
		Key:                      key,
		ConsistentRead:           aws.Bool(input.ConsistentRead),
		ProjectionExpression:     optionalString(input.ProjectionExpression),
		ExpressionAttributeNames: input.ExpressionAttributeNames,
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_get_started", "table", strings.TrimSpace(input.TableName))
	response, err := client.GetItem(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_dynamodb_get_failed", "table", strings.TrimSpace(input.TableName), "error", err)
		return types.DynamoDBGetOutput{}, types.DynamoDBError{Operation: "get_item", Err: err}
	}
	if len(response.Item) == 0 {
		return types.DynamoDBGetOutput{Found: false}, nil
	}
	if input.Target != nil {
		if err := attributevalue.UnmarshalMap(response.Item, input.Target); err != nil {
			return types.DynamoDBGetOutput{}, types.DecodeError{Operation: "dynamodb_get_item", Err: err}
		}
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_get_completed", "table", strings.TrimSpace(input.TableName), "found", true)
	return types.DynamoDBGetOutput{Found: true, Item: response.Item}, nil
}

// QueryDynamoDB retrieves items from a DynamoDB table by key condition expression.
func (c *Consumer) QueryDynamoDB(ctx context.Context, input types.DynamoDBQueryInput) (types.DynamoDBQueryOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.dynamoDBClient(ctx)
	if err != nil {
		return types.DynamoDBQueryOutput{}, types.DynamoDBError{Operation: "load_config", Err: err}
	}
	values, err := marshalDynamoValues(input.ExpressionAttributeValues)
	if err != nil {
		return types.DynamoDBQueryOutput{}, types.DynamoDBError{Operation: "marshal_expression_values", Err: err}
	}

	request := &dynamodb.QueryInput{
		TableName:                 aws.String(strings.TrimSpace(input.TableName)),
		KeyConditionExpression:    optionalString(input.KeyConditionExpression),
		FilterExpression:          optionalString(input.FilterExpression),
		ProjectionExpression:      optionalString(input.ProjectionExpression),
		ExpressionAttributeNames:  input.ExpressionAttributeNames,
		ExpressionAttributeValues: values,
		IndexName:                 optionalString(input.IndexName),
		Limit:                     optionalInt32(input.Limit),
		ScanIndexForward:          input.ScanIndexForward,
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_query_started", "table", strings.TrimSpace(input.TableName))
	response, err := client.Query(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_dynamodb_query_failed", "table", strings.TrimSpace(input.TableName), "error", err)
		return types.DynamoDBQueryOutput{}, types.DynamoDBError{Operation: "query", Err: err}
	}
	if input.Target != nil {
		if err := attributevalue.UnmarshalListOfMaps(response.Items, input.Target); err != nil {
			return types.DynamoDBQueryOutput{}, types.DecodeError{Operation: "dynamodb_query_items", Err: err}
		}
	}
	c.logger.InfoContext(ctx, "consumer_dynamodb_query_completed", "table", strings.TrimSpace(input.TableName), "count", response.Count)
	return types.DynamoDBQueryOutput{Count: response.Count, Items: response.Items}, nil
}

func marshalDynamoMap(value any) (map[string]dynamodbtypes.AttributeValue, error) {
	if value == nil {
		return map[string]dynamodbtypes.AttributeValue{}, nil
	}
	if raw, ok := value.(map[string]dynamodbtypes.AttributeValue); ok {
		return raw, nil
	}
	return attributevalue.MarshalMap(value)
}

func marshalDynamoValues(values map[string]any) (map[string]dynamodbtypes.AttributeValue, error) {
	if len(values) == 0 {
		return nil, nil
	}
	marshaled := make(map[string]dynamodbtypes.AttributeValue, len(values))
	for key, value := range values {
		if raw, ok := value.(dynamodbtypes.AttributeValue); ok {
			marshaled[key] = raw
			continue
		}
		attribute, err := attributevalue.Marshal(value)
		if err != nil {
			return nil, err
		}
		marshaled[key] = attribute
	}
	return marshaled, nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return aws.String(trimmed)
}

func optionalInt32(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	return aws.Int32(value)
}
