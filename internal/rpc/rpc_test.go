package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallSuccess(t *testing.T) {
	var gotMethod string
	var gotParams map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		gotMethod, gotParams = req.Method, req.Params
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": "pong",
		})
	}))
	defer srv.Close()

	var result string
	err := NewClient(srv.URL).Call(context.Background(), "JSONRPC.Ping",
		map[string]any{"echo": true}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "JSONRPC.Ping" || gotParams["echo"] != true || result != "pong" {
		t.Fatalf("method=%q params=%v result=%q", gotMethod, gotParams, result)
	}
}

func TestCallRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"error": map[string]any{"code": -32601, "message": "Method not found."},
		})
	}))
	defer srv.Close()

	var result any
	err := NewClient(srv.URL).Call(context.Background(), "Bogus.Method", nil, &result)
	rpcErr, ok := err.(*RPCError)
	if !ok || rpcErr.Code != -32601 {
		t.Fatalf("expected *RPCError(-32601), got %T %v", err, err)
	}
}
