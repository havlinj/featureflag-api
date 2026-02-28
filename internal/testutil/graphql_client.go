package testutil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"github.com/jan-havlin-dev/featureflag-api/transport/graphql"
)

// GraphQLClient sends GraphQL requests over HTTP(S). For HTTPS with self-signed certs
// use NewClientWithTLS or ensure the http.Client uses InsecureSkipVerify.
type GraphQLClient struct {
	URL       string
	AuthToken string
	Client    *http.Client
}

// NewClient creates a client for the given URL and optional auth token.
// The default client uses TLS with InsecureSkipVerify (suitable for tests).
func NewClient(url, authToken string) *GraphQLClient {
	return &GraphQLClient{
		URL:       url,
		AuthToken: authToken,
		Client:    defaultHTTPClient(),
	}
}

// NewClientForIntegration creates a client for integration tests: baseURL is the
// full base URL (e.g. "https://127.0.0.1:port") and the client skips TLS verify.
func NewClientForIntegration(baseURL string) *GraphQLClient {
	return NewClient(baseURL, "")
}

// SetToken sets the Bearer token for subsequent requests.
func (c *GraphQLClient) SetToken(token string) {
	c.AuthToken = token
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// DoRequest sends a GraphQL request and returns the response (data and/or errors).
func (c *GraphQLClient) DoRequest(query string, variables map[string]interface{}) (*graphql.GraphQLResponse, error) {
	reqBody := graphql.GraphQLRequest{
		Query:     query,
		Variables: variables,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var gqlResp graphql.GraphQLResponse
	if err := json.Unmarshal(respBytes, &gqlResp); err != nil {
		return nil, err
	}
	return &gqlResp, nil
}
