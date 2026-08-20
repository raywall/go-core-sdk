// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public REST contracts for the consumer service.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"encoding/json"
	"net/http"
)

// AuthorizationToken exposes the string representation used in an Authorization header.
type AuthorizationToken interface {
	// ToString returns the complete Authorization header value, including token type.
	ToString() string
}

// TokenProvider exposes a stable authorization token instance.
type TokenProvider interface {
	// Token returns the current token object.
	Token() AuthorizationToken
}

// RESTRequest describes an outbound REST call.
type RESTRequest struct {
	// Method is the HTTP method used for the call.
	Method string
	// URL is the absolute endpoint URL.
	URL string
	// Headers contains request headers. Header names are handled by net/http.
	Headers map[string]string
	// Query contains query string parameters appended to URL.
	Query map[string]string
	// Body is the request payload. Supported values include nil, []byte, string,
	// io.Reader and JSON-marshalable values.
	Body any
	// UseToken controls whether the consumer injects the Authorization header
	// from the configured TokenProvider.
	UseToken bool
}

// RESTResponse contains the full REST response body and metadata.
type RESTResponse struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
	// Status is the raw HTTP status text returned by the server.
	Status string
	// Headers contains the response headers.
	Headers http.Header
	// Body contains the full response body.
	Body []byte
}

// DecodeJSON decodes Body into target.
func (r RESTResponse) DecodeJSON(target any) error {
	if err := json.Unmarshal(r.Body, target); err != nil {
		return DecodeError{Operation: "rest_json", Err: err}
	}
	return nil
}
