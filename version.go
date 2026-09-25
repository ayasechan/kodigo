package kodi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ayasechan/kodigo/internal/rpc"
)

// APIVersion is a Kodi JSON-RPC API version (see JSONRPC.Version):
//   - Major is bumped on backwards incompatible changes.
//   - Minor is bumped on backwards compatible additions.
//   - Patch covers internal changes only and is ignored by negotiation.
type APIVersion struct {
	Major int
	Minor int
	Patch int
}

// ParseAPIVersion parses "major.minor.patch" (patch optional).
func ParseAPIVersion(s string) (APIVersion, error) {
	var v APIVersion
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return v, fmt.Errorf("kodi: invalid version %q", s)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("kodi: invalid version %q", s)
		}
		nums[i] = n
	}
	return APIVersion{Major: nums[0], Minor: nums[1], Patch: nums[2]}, nil
}

func mustParseAPIVersion(s string) APIVersion {
	v, err := ParseAPIVersion(s)
	if err != nil {
		panic(fmt.Sprintf("kodi: invalid version %q: %v", s, err))
	}
	return v
}

func (v APIVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares major, then minor, then patch.
func (v APIVersion) Compare(o APIVersion) int {
	if v.Major != o.Major {
		return cmpInt(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return cmpInt(v.Minor, o.Minor)
	}
	return cmpInt(v.Patch, o.Patch)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Compatibility is the outcome of negotiating with a server.
type Compatibility int

const (
	// Compatible means the server speaks the same major version and its
	// minor is >= the client's: the client only uses a subset the server
	// understands.
	Compatible Compatibility = iota
	// ServerOlderMinor means the same major but the server's minor is older
	// than the schema the client was generated from. Calls still work, but
	// fields/methods added after the server's minor will be missing.
	ServerOlderMinor
	// IncompatibleMajor means the major versions differ: the API definition
	// is not backwards compatible, do not use this client version.
	IncompatibleMajor
)

func (c Compatibility) String() string {
	switch c {
	case Compatible:
		return "compatible"
	case ServerOlderMinor:
		return "server-older-minor"
	case IncompatibleMajor:
		return "incompatible-major"
	default:
		return "unknown"
	}
}

// Negotiation is the result of probing a server's JSONRPC.Version.
type Negotiation struct {
	Server        APIVersion
	Client        APIVersion
	Compatibility Compatibility
}

// Err returns nil unless the major versions differ.
func (n Negotiation) Err() error {
	if n.Compatibility == IncompatibleMajor {
		return fmt.Errorf("kodi: server API version %s is incompatible with client version %s (major version differs)",
			n.Server, n.Client)
	}
	return nil
}

// Check compares a server version against a client version. It is pure and
// version-agnostic: pass any generated version's schema version as client.
func Check(server, client APIVersion) Negotiation {
	n := Negotiation{Server: server, Client: client}
	switch {
	case server.Major != client.Major:
		n.Compatibility = IncompatibleMajor
	case server.Minor < client.Minor:
		n.Compatibility = ServerOlderMinor
	default:
		n.Compatibility = Compatible
	}
	return n
}

// ProbeServerVersion queries the server's JSONRPC.Version with a raw call,
// so it works before any version is selected.
func ProbeServerVersion(ctx context.Context, c rpc.Caller) (APIVersion, error) {
	var res struct {
		Version struct {
			Major int `json:"major"`
			Minor int `json:"minor"`
			Patch int `json:"patch"`
		} `json:"version"`
	}
	if err := c.Call(ctx, "JSONRPC.Version", nil, &res); err != nil {
		return APIVersion{}, fmt.Errorf("kodi: probe server version: %w", err)
	}
	return APIVersion{Major: res.Version.Major, Minor: res.Version.Minor, Patch: res.Version.Patch}, nil
}

// Negotiate probes the server version and checks it against the default
// (latest supported) client version. To negotiate against a specific
// version, combine ProbeServerVersion with Check.
func Negotiate(ctx context.Context, c rpc.Caller) (Negotiation, error) {
	server, err := ProbeServerVersion(ctx, c)
	if err != nil {
		return Negotiation{}, err
	}
	return Check(server, SupportedAPIVersion), nil
}

// DefaultNegotiateTimeout bounds the version probe when ctx has no deadline.
const DefaultNegotiateTimeout = 10 * time.Second
