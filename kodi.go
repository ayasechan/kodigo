package kodi

import (
	"context"

	"github.com/ayasechan/kodigo/internal/rpc"
	"github.com/ayasechan/kodigo/v12"
	"github.com/ayasechan/kodigo/v13"
)

// SchemaVersion is the schema version of the default API client.
const SchemaVersion = v13.SchemaVersion

// SupportedAPIVersion is the API version of the default client.
var SupportedAPIVersion = mustParseAPIVersion(v13.SchemaVersion)

// SupportedVersions lists the API versions with generated clients,
// newest first. Add new versions here as version packages land.
func SupportedVersions() []APIVersion {
	return []APIVersion{
		SupportedAPIVersion,
		mustParseAPIVersion(v12.SchemaVersion),
	}
}

// Client is the default JSON-RPC client: an alias for the latest supported
// versioned client, so all typed methods are available directly.
type Client = v13.Client

// Option configures a Client. Alias for the shared transport option.
type Option = rpc.Option

// WithBasicAuth sets HTTP basic auth credentials (Kodi webserver user/password).
var WithBasicAuth = rpc.WithBasicAuth

// WithHTTPClient overrides the default *http.Client.
var WithHTTPClient = rpc.WithHTTPClient

// RPCError is a JSON-RPC 2.0 error object returned by the server.
type RPCError = rpc.RPCError

// NewClient creates the default client targeting endpoint
// (e.g. "http://localhost:8080/jsonrpc").
func NewClient(endpoint string, opts ...rpc.Option) *Client {
	return v13.NewClient(endpoint, opts...)
}

// NegotiateClient creates the default client and probes the server version
// in one step. When the major versions differ the client is still returned
// (for raw Call use) but err is non-nil.
func NegotiateClient(ctx context.Context, endpoint string, opts ...rpc.Option) (*Client, Negotiation, error) {
	c := NewClient(endpoint, opts...)
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultNegotiateTimeout)
		defer cancel()
	}
	n, err := Negotiate(ctx, c)
	if err != nil {
		return c, n, err
	}
	return c, n, n.Err()
}
