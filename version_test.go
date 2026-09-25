package kodi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ayasechan/kodigo/internal/rpc"
	"github.com/ayasechan/kodigo/v12"
)

func TestParseAPIVersion(t *testing.T) {
	cases := []struct {
		in      string
		want    APIVersion
		wantErr bool
	}{
		{"13.5.0", APIVersion{13, 5, 0}, false},
		{"12.5", APIVersion{12, 5, 0}, false},
		{" 13.5.0 ", APIVersion{13, 5, 0}, false},
		{"13", APIVersion{}, true},
		{"13.x.0", APIVersion{}, true},
		{"13.-1.0", APIVersion{}, true},
		{"", APIVersion{}, true},
	}
	for _, c := range cases {
		got, err := ParseAPIVersion(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseAPIVersion(%q): expected error", c.in)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("ParseAPIVersion(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
}

// versionServer returns a fake Kodi server reporting the given version.
func versionServer(t *testing.T, major, minor, patch int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]any{
				"version": map[string]any{"major": major, "minor": minor, "patch": patch},
			},
		})
	}))
}

func TestCheck(t *testing.T) {
	client := APIVersion{13, 14, 0}
	cases := []struct {
		name   string
		server APIVersion
		want   Compatibility
	}{
		{"exact", APIVersion{13, 14, 0}, Compatible},
		{"server newer minor", APIVersion{13, 15, 0}, Compatible},
		{"server newer patch", APIVersion{13, 14, 99}, Compatible},
		{"server older minor", APIVersion{13, 10, 0}, ServerOlderMinor},
		{"server older major", APIVersion{12, 99, 0}, IncompatibleMajor},
		{"server newer major", APIVersion{14, 0, 0}, IncompatibleMajor},
	}
	for _, c := range cases {
		n := Check(c.server, client)
		if n.Compatibility != c.want {
			t.Errorf("Check(%s): = %s, want %s", c.server, n.Compatibility, c.want)
		}
		if (n.Err() != nil) != (c.want == IncompatibleMajor) {
			t.Errorf("Check(%s).Err() = %v", c.server, n.Err())
		}
	}
}

func TestNegotiateAgainstSupported(t *testing.T) {
	sv := SupportedAPIVersion
	srv := versionServer(t, sv.Major, sv.Minor, 0)
	defer srv.Close()

	n, err := Negotiate(context.Background(), rpc.NewClient(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if n.Server != sv || n.Client != sv || n.Compatibility != Compatible {
		t.Fatalf("unexpected negotiation: %+v", n)
	}
}

func TestNegotiateClient(t *testing.T) {
	sv := SupportedAPIVersion

	srv := versionServer(t, sv.Major, sv.Minor, 0)
	defer srv.Close()
	c, n, err := NegotiateClient(context.Background(), srv.URL)
	if err != nil || c == nil || n.Compatibility != Compatible {
		t.Fatalf("got client=%v negotiation=%+v err=%v", c, n, err)
	}

	old := versionServer(t, sv.Major-1, 0, 0)
	defer old.Close()
	_, _, err = NegotiateClient(context.Background(), old.URL)
	if err == nil {
		t.Fatal("expected major-mismatch error")
	}
	t.Logf("mismatch error: %v", err)
}

// TestVersionSwitchV12 proves the multi-version flow against a real v12
// server: the default negotiation refuses, the v12 client matches, and Wrap
// reuses the probed transport.
func TestVersionSwitchV12(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result any = map[string]any{
			"version": map[string]any{"major": 12, "minor": 4, "patch": 0},
		}
		if req.Method == "JSONRPC.Ping" {
			result = "pong"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": result,
		})
	}))
	defer srv.Close()

	ctx := context.Background()
	raw := rpc.NewClient(srv.URL)

	n, err := Negotiate(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	if n.Compatibility != IncompatibleMajor {
		t.Fatalf("default client vs v12 server: = %s, want incompatible-major", n.Compatibility)
	}

	var v12ver APIVersion
	for _, v := range SupportedVersions() {
		if v.Major == n.Server.Major {
			v12ver = v
		}
	}
	if v12ver.Major != v12.APIMajor {
		t.Fatalf("SupportedVersions() missing v12: %v", SupportedVersions())
	}
	if got := Check(n.Server, v12ver); got.Compatibility != Compatible {
		t.Fatalf("v12 client vs v12 server: = %s, want compatible", got.Compatibility)
	}
	if pong, err := v12.Wrap(raw).JSONRPCPing(ctx); err != nil || pong != "pong" {
		t.Fatalf("v12 Ping = %q, %v", pong, err)
	}
}
