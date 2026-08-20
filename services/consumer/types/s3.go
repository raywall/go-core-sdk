// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public S3 request and response contracts.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

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
