// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public Secrets Manager request and response contracts.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

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
