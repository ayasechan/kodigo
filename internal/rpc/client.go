package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

// Caller performs a single JSON-RPC request.
//
// *Client implements it, and so does every versioned client (via the
// embedded *Client), so version-agnostic helpers such as negotiation accept
// any of them.
type Caller interface {
	Call(ctx context.Context, method string, params, result any) error
}

var _ Caller = (*Client)(nil)

// RPCError is a JSON-RPC 2.0 error object returned by the server.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("jsonrpc: code=%d message=%s", e.Code, e.Message)
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint64      `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Option configures a Client.
type Option func(*Client)

// WithBasicAuth sets HTTP basic auth credentials (Kodi webserver user/password).
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		c.Username = username
		c.Password = password
	}
}

// WithHTTPClient overrides the default *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		c.HTTP = h
	}
}

// Client is a JSON-RPC 2.0 client speaking to Kodi's webserver
// (default endpoint http://<host>:8080/jsonrpc).
//
// It is version-agnostic: versioned packages (v13, …) embed it and add
// typed methods for their API version.
type Client struct {
	Endpoint string
	Username string
	Password string
	HTTP     *http.Client

	id atomic.Uint64
}

// NewClient creates a Client targeting endpoint (e.g. "http://localhost:8080/jsonrpc").
func NewClient(endpoint string, opts ...Option) *Client {
	c := &Client{Endpoint: endpoint, HTTP: http.DefaultClient}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Call performs a single JSON-RPC request. params may be nil, a struct, or a
// map. If result is non-nil, the response result is unmarshalled into it.
func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.id.Add(1),
	})
	if err != nil {
		return fmt.Errorf("jsonrpc: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("jsonrpc: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("jsonrpc: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jsonrpc: unexpected http status %s", resp.Status)
	}

	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("jsonrpc: decode response: %w", err)
	}
	if out.Error != nil {
		return out.Error
	}
	if result != nil && len(out.Result) > 0 {
		if err := json.Unmarshal(out.Result, result); err != nil {
			return fmt.Errorf("jsonrpc: decode result: %w", err)
		}
	}
	return nil
}
