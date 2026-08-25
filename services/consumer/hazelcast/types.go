// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements public contracts for distributed configuration reads.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"context"

	hz "github.com/hazelcast/hazelcast-go-client"
)

// SourceKind identifies where Hazelcast client configuration JSON is stored.
type SourceKind string

const (
	// SourceDefault uses the zero value Hazelcast client configuration.
	SourceDefault SourceKind = ""
	// SourceInline loads Hazelcast client configuration from Source.Data.
	SourceInline SourceKind = "inline"
	// SourceFile loads Hazelcast client configuration from a local filesystem path.
	SourceFile SourceKind = "file"
	// SourceS3 loads Hazelcast client configuration from an S3 object.
	SourceS3 SourceKind = "s3"
	// SourceSecretsManager loads Hazelcast client configuration from a Secrets Manager secret.
	SourceSecretsManager SourceKind = "secretsmanager"
)

// Source describes where Hazelcast client configuration JSON should be loaded from.
type Source struct {
	// Kind identifies the source backend. The zero value uses the default Hazelcast config.
	Kind SourceKind
	// Data contains inline JSON when Kind is SourceInline.
	Data []byte
	// Path is the local file path when Kind is SourceFile.
	Path string
	// Bucket is the S3 bucket when Kind is SourceS3.
	Bucket string
	// Key is the S3 object key when Kind is SourceS3.
	Key string
	// SecretID is the Secrets Manager secret name or ARN when Kind is SourceSecretsManager.
	SecretID string
	// VersionID optionally selects a Secrets Manager secret version.
	VersionID string
	// VersionStage optionally selects a Secrets Manager version stage.
	VersionStage string
	// AWSRegion optionally pins the AWS region used by default S3 or Secrets Manager clients.
	AWSRegion string
}

// Config controls Hazelcast consumer construction.
type Config struct {
	// Source optionally provides JSON used to build HazelcastConfig.
	Source Source
	// HazelcastConfig is used directly when Source is empty, or as the base
	// configuration when Source JSON is loaded.
	HazelcastConfig hz.Config
	// DefaultMap is used by read methods when mapName is empty.
	DefaultMap string
}

// Value contains a value read from a Hazelcast map.
type Value struct {
	// MapName is the map used for the read operation.
	MapName string
	// Key is the key requested from the map.
	Key any
	// Raw is the value returned by Hazelcast.
	Raw any
	// Found indicates whether the key exists in the map.
	Found bool
}

// Map defines the Hazelcast map operations used by Client.
type Map interface {
	// ContainsKey reports whether key exists in the map.
	ContainsKey(context.Context, interface{}) (bool, error)
	// Get retrieves a value by key.
	Get(context.Context, interface{}) (interface{}, error)
}

// MapProvider defines the Hazelcast client operations used by Client.
type MapProvider interface {
	// GetMap returns a distributed map by name.
	GetMap(context.Context, string) (Map, error)
	// Shutdown closes the underlying provider.
	Shutdown(context.Context) error
}

// Starter starts a Hazelcast-backed map provider from a Hazelcast config.
type Starter func(context.Context, hz.Config) (MapProvider, error)

// SourceLoader loads raw Hazelcast client configuration JSON from Source.
type SourceLoader interface {
	// Load reads source bytes.
	Load(context.Context, Source) ([]byte, error)
}
