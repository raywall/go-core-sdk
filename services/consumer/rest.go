// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements outbound REST calls.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// RequestBuilder builds and executes an outbound REST request.
type RequestBuilder struct {
	consumer *Consumer
	request  types.RESTRequest
}

// REST creates a request builder for an outbound REST call.
func (c *Consumer) REST(method string, url string) *RequestBuilder {
	return &RequestBuilder{
		consumer: c,
		request: types.RESTRequest{
			Method:  method,
			URL:     url,
			Headers: map[string]string{},
			Query:   map[string]string{},
		},
	}
}

// WithHeader adds or replaces a request header.
func (b *RequestBuilder) WithHeader(key string, value string) *RequestBuilder {
	b.request.Headers[key] = value
	return b
}

// WithHeaders adds or replaces multiple request headers.
func (b *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	for key, value := range headers {
		b.request.Headers[key] = value
	}
	return b
}

// WithQueryParam adds or replaces a query string parameter.
func (b *RequestBuilder) WithQueryParam(key string, value string) *RequestBuilder {
	b.request.Query[key] = value
	return b
}

// WithBody sets the request payload.
func (b *RequestBuilder) WithBody(body any) *RequestBuilder {
	b.request.Body = body
	return b
}

// WithToken enables Authorization header injection from the configured token provider.
func (b *RequestBuilder) WithToken() *RequestBuilder {
	b.request.UseToken = true
	return b
}

// Do executes the built REST request.
func (b *RequestBuilder) Do(ctx context.Context) (types.RESTResponse, error) {
	return b.consumer.DoREST(ctx, b.request)
}

// DoREST executes a REST request described by input.
func (c *Consumer) DoREST(ctx context.Context, input types.RESTRequest) (types.RESTResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return types.RESTResponse{}, err
	}

	requestBody, contentType, err := encodeRESTBody(input.Body)
	if err != nil {
		return types.RESTResponse{}, types.RESTError{Operation: "encode_body", Err: err}
	}

	method := strings.TrimSpace(input.Method)
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSpace(input.URL), requestBody)
	if err != nil {
		return types.RESTResponse{}, types.RESTError{Operation: "build_request", Err: err}
	}

	for key, value := range input.Headers {
		req.Header.Set(key, value)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range input.Query {
		query := req.URL.Query()
		query.Set(key, value)
		req.URL.RawQuery = query.Encode()
	}
	if input.UseToken {
		authorization, err := c.authorizationHeader()
		if err != nil {
			return types.RESTResponse{}, err
		}
		req.Header.Set("Authorization", authorization)
	}

	c.logger.InfoContext(ctx, "consumer_rest_request_started", "method", req.Method, "url", sanitizeURL(req.URL.String()), "with_token", input.UseToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_rest_request_failed", "method", req.Method, "url", sanitizeURL(req.URL.String()), "error", err)
		return types.RESTResponse{}, types.RESTError{Operation: "execute", Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.RESTResponse{}, types.RESTError{Operation: "read_response", Err: err}
	}

	c.logger.InfoContext(ctx, "consumer_rest_request_completed", "method", req.Method, "url", sanitizeURL(req.URL.String()), "status_code", resp.StatusCode)
	return types.RESTResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}

func (c *Consumer) authorizationHeader() (string, error) {
	if c.tokenProvider == nil {
		return "", types.TokenRequiredError{Reason: "token provider is not configured"}
	}
	token := c.tokenProvider.Token()
	if token == nil {
		return "", types.TokenRequiredError{Reason: "token provider returned nil token"}
	}
	authorization := strings.TrimSpace(token.ToString())
	if authorization == "" {
		return "", types.TokenRequiredError{Reason: "token value is empty"}
	}
	return authorization, nil
}

func encodeRESTBody(body any) (io.Reader, string, error) {
	switch value := body.(type) {
	case nil:
		return nil, "", nil
	case io.Reader:
		return value, "", nil
	case []byte:
		return bytes.NewReader(value), "", nil
	case string:
		return strings.NewReader(value), "", nil
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(payload), "application/json", nil
	}
}

func sanitizeURL(value string) string {
	if idx := strings.Index(value, "?"); idx >= 0 {
		return value[:idx]
	}
	return value
}
