// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// rest implements the outbound REST client.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// Client executes outbound REST requests.
//
// Client is safe for concurrent use when the configured HTTP client and token
// provider are safe for concurrent use.
type Client struct {
	config        Config
	logger        *slog.Logger
	httpClient    *http.Client
	tokenProvider TokenProvider
}

// New constructs a REST Client from Config.
func New(config Config, configurers ...Option) (*Client, error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: normalized.HTTPTimeout}
	}

	return &Client{
		config:        normalized,
		logger:        options.logger,
		httpClient:    httpClient,
		tokenProvider: options.tokenProvider,
	}, nil
}

// Config returns a copy of the normalized REST configuration.
func (c *Client) Config() Config {
	if c == nil {
		return Config{}
	}
	return c.config
}

// RequestBuilder builds and executes an outbound REST request.
type RequestBuilder struct {
	client  *Client
	request Request
}

// REST creates a request builder for an outbound REST call.
func (c *Client) REST(method string, url string) *RequestBuilder {
	return &RequestBuilder{
		client: c,
		request: Request{
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
func (b *RequestBuilder) Do(ctx context.Context) (Response, error) {
	return b.client.Do(ctx, b.request)
}

// Do executes a REST request described by input.
func (c *Client) Do(ctx context.Context, input Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	requestBody, contentType, err := encodeBody(input.Body)
	if err != nil {
		return Response{}, RESTError{Operation: "encode_body", Err: err}
	}

	method := strings.TrimSpace(input.Method)
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSpace(input.URL), requestBody)
	if err != nil {
		return Response{}, RESTError{Operation: "build_request", Err: err}
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
			return Response{}, err
		}
		req.Header.Set("Authorization", authorization)
	}

	c.logger.InfoContext(ctx, "consumer_rest_request_started", "method", req.Method, "url", sanitizeURL(req.URL.String()), "with_token", input.UseToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_rest_request_failed", "method", req.Method, "url", sanitizeURL(req.URL.String()), "error", err)
		return Response{}, RESTError{Operation: "execute", Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, RESTError{Operation: "read_response", Err: err}
	}

	c.logger.InfoContext(ctx, "consumer_rest_request_completed", "method", req.Method, "url", sanitizeURL(req.URL.String()), "status_code", resp.StatusCode)
	return Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}

func (c *Client) authorizationHeader() (string, error) {
	if c.tokenProvider == nil {
		return "", TokenRequiredError{Reason: "token provider is not configured"}
	}
	token := c.tokenProvider.Token()
	if token == nil {
		return "", TokenRequiredError{Reason: "token provider returned nil token"}
	}
	authorization := strings.TrimSpace(token.ToString())
	if authorization == "" {
		return "", TokenRequiredError{Reason: "token value is empty"}
	}
	return authorization, nil
}

func encodeBody(body any) (io.Reader, string, error) {
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
