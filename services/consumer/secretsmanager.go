// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements Secrets Manager convenience operations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// GetSecret retrieves a secret value from AWS Secrets Manager.
func (c *Consumer) GetSecret(ctx context.Context, input types.SecretGetInput) (types.SecretGetOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return types.SecretGetOutput{}, types.SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretID),
		VersionId:    optionalString(input.VersionID),
		VersionStage: optionalString(input.VersionStage),
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_get_started", "secret_id", secretID, "version_stage", strings.TrimSpace(input.VersionStage))
	response, err := client.GetSecretValue(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_secretsmanager_get_failed", "secret_id", secretID, "error", err)
		return types.SecretGetOutput{}, types.SecretsManagerError{Operation: "get_secret_value", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_get_completed", "secret_id", secretID, "name", aws.ToString(response.Name), "version_id", aws.ToString(response.VersionId))
	return types.SecretGetOutput{
		ARN:           aws.ToString(response.ARN),
		Name:          aws.ToString(response.Name),
		VersionID:     aws.ToString(response.VersionId),
		VersionStages: append([]string(nil), response.VersionStages...),
		SecretString:  aws.ToString(response.SecretString),
		SecretBinary:  append([]byte(nil), response.SecretBinary...),
	}, nil
}

// GetSecretJSON retrieves a secret and decodes its value as JSON into target.
func (c *Consumer) GetSecretJSON(ctx context.Context, input types.SecretGetInput, target any) (types.SecretGetOutput, error) {
	output, err := c.GetSecret(ctx, input)
	if err != nil {
		return types.SecretGetOutput{}, err
	}
	if err := json.Unmarshal(output.Bytes(), target); err != nil {
		return types.SecretGetOutput{}, types.DecodeError{Operation: "secretsmanager_json", Err: err}
	}
	return output, nil
}

// PutSecret stores a new value version for an existing secret.
func (c *Consumer) PutSecret(ctx context.Context, input types.SecretPutInput) (types.SecretPutOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return types.SecretPutOutput{}, types.SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.PutSecretValueInput{
		SecretId:           aws.String(secretID),
		SecretString:       secretStringPointer(input.SecretString, input.SecretBinary),
		SecretBinary:       copySecretBinary(input.SecretBinary, input.SecretString),
		ClientRequestToken: optionalString(input.ClientRequestToken),
		VersionStages:      append([]string(nil), input.VersionStages...),
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_put_started", "secret_id", secretID, "version_stages", len(input.VersionStages))
	response, err := client.PutSecretValue(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_secretsmanager_put_failed", "secret_id", secretID, "error", err)
		return types.SecretPutOutput{}, types.SecretsManagerError{Operation: "put_secret_value", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_put_completed", "secret_id", secretID, "name", aws.ToString(response.Name), "version_id", aws.ToString(response.VersionId))
	return types.SecretPutOutput{
		ARN:           aws.ToString(response.ARN),
		Name:          aws.ToString(response.Name),
		VersionID:     aws.ToString(response.VersionId),
		VersionStages: append([]string(nil), response.VersionStages...),
	}, nil
}

// UpdateSecret updates secret metadata and optionally writes a new value.
func (c *Consumer) UpdateSecret(ctx context.Context, input types.SecretUpdateInput) (types.SecretUpdateOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return types.SecretUpdateOutput{}, types.SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.UpdateSecretInput{
		SecretId:           aws.String(secretID),
		Description:        optionalString(input.Description),
		KmsKeyId:           optionalString(input.KMSKeyID),
		SecretString:       secretStringPointer(input.SecretString, input.SecretBinary),
		SecretBinary:       copySecretBinary(input.SecretBinary, input.SecretString),
		ClientRequestToken: optionalString(input.ClientRequestToken),
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_update_started", "secret_id", secretID, "has_value", input.SecretString != "" || len(input.SecretBinary) > 0)
	response, err := client.UpdateSecret(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_secretsmanager_update_failed", "secret_id", secretID, "error", err)
		return types.SecretUpdateOutput{}, types.SecretsManagerError{Operation: "update_secret", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_update_completed", "secret_id", secretID, "name", aws.ToString(response.Name), "version_id", aws.ToString(response.VersionId))
	return types.SecretUpdateOutput{
		ARN:       aws.ToString(response.ARN),
		Name:      aws.ToString(response.Name),
		VersionID: aws.ToString(response.VersionId),
	}, nil
}

func secretStringPointer(secretString string, secretBinary []byte) *string {
	if secretString == "" {
		return nil
	}
	return aws.String(secretString)
}

func copySecretBinary(secretBinary []byte, secretString string) []byte {
	if secretString != "" || len(secretBinary) == 0 {
		return nil
	}
	return append([]byte(nil), secretBinary...)
}
