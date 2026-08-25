// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws implements Secrets Manager convenience operations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws

import (
	"context"
	"encoding/json"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// GetSecret retrieves a secret value from AWS Secrets Manager.
func (c *Client) GetSecret(ctx context.Context, input SecretGetInput) (SecretGetOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return SecretGetOutput{}, SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.GetSecretValueInput{
		SecretId:     awssdk.String(secretID),
		VersionId:    optionalString(input.VersionID),
		VersionStage: optionalString(input.VersionStage),
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_get_started", "secret_id", secretID, "version_stage", strings.TrimSpace(input.VersionStage))
	response, err := client.GetSecretValue(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_secretsmanager_get_failed", "secret_id", secretID, "error", err)
		return SecretGetOutput{}, SecretsManagerError{Operation: "get_secret_value", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_get_completed", "secret_id", secretID, "name", awssdk.ToString(response.Name), "version_id", awssdk.ToString(response.VersionId))
	return SecretGetOutput{
		ARN:           awssdk.ToString(response.ARN),
		Name:          awssdk.ToString(response.Name),
		VersionID:     awssdk.ToString(response.VersionId),
		VersionStages: append([]string(nil), response.VersionStages...),
		SecretString:  awssdk.ToString(response.SecretString),
		SecretBinary:  append([]byte(nil), response.SecretBinary...),
	}, nil
}

// GetSecretJSON retrieves a secret and decodes its value as JSON into target.
func (c *Client) GetSecretJSON(ctx context.Context, input SecretGetInput, target any) (SecretGetOutput, error) {
	output, err := c.GetSecret(ctx, input)
	if err != nil {
		return SecretGetOutput{}, err
	}
	if err := json.Unmarshal(output.Bytes(), target); err != nil {
		return SecretGetOutput{}, DecodeError{Operation: "secretsmanager_json", Err: err}
	}
	return output, nil
}

// PutSecret stores a new value version for an existing secret.
func (c *Client) PutSecret(ctx context.Context, input SecretPutInput) (SecretPutOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return SecretPutOutput{}, SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.PutSecretValueInput{
		SecretId:           awssdk.String(secretID),
		SecretString:       secretStringPointer(input.SecretString, input.SecretBinary),
		SecretBinary:       copySecretBinary(input.SecretBinary, input.SecretString),
		ClientRequestToken: optionalString(input.ClientRequestToken),
		VersionStages:      append([]string(nil), input.VersionStages...),
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_put_started", "secret_id", secretID, "version_stages", len(input.VersionStages))
	response, err := client.PutSecretValue(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_secretsmanager_put_failed", "secret_id", secretID, "error", err)
		return SecretPutOutput{}, SecretsManagerError{Operation: "put_secret_value", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_put_completed", "secret_id", secretID, "name", awssdk.ToString(response.Name), "version_id", awssdk.ToString(response.VersionId))
	return SecretPutOutput{
		ARN:           awssdk.ToString(response.ARN),
		Name:          awssdk.ToString(response.Name),
		VersionID:     awssdk.ToString(response.VersionId),
		VersionStages: append([]string(nil), response.VersionStages...),
	}, nil
}

// UpdateSecret updates secret metadata and optionally writes a new value.
func (c *Client) UpdateSecret(ctx context.Context, input SecretUpdateInput) (SecretUpdateOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.secretsManagerClient(ctx)
	if err != nil {
		return SecretUpdateOutput{}, SecretsManagerError{Operation: "load_config", Err: err}
	}

	secretID := strings.TrimSpace(input.SecretID)
	request := &secretsmanager.UpdateSecretInput{
		SecretId:           awssdk.String(secretID),
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
		return SecretUpdateOutput{}, SecretsManagerError{Operation: "update_secret", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_secretsmanager_update_completed", "secret_id", secretID, "name", awssdk.ToString(response.Name), "version_id", awssdk.ToString(response.VersionId))
	return SecretUpdateOutput{
		ARN:       awssdk.ToString(response.ARN),
		Name:      awssdk.ToString(response.Name),
		VersionID: awssdk.ToString(response.VersionId),
	}, nil
}

func secretStringPointer(secretString string, secretBinary []byte) *string {
	if secretString == "" {
		return nil
	}
	return awssdk.String(secretString)
}

func copySecretBinary(secretBinary []byte, secretString string) []byte {
	if secretString != "" || len(secretBinary) == 0 {
		return nil
	}
	return append([]byte(nil), secretBinary...)
}
