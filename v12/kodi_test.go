package v12

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnumValues(t *testing.T) {
	if PlayerRepeatOff != "off" || PlayerRepeatOne != "one" || PlayerRepeatAll != "all" {
		t.Fatalf("unexpected PlayerRepeat consts: %q %q %q", PlayerRepeatOff, PlayerRepeatOne, PlayerRepeatAll)
	}
	if MethodPlayerOpen != "Player.Open" || MethodJSONRPCPing != "JSONRPC.Ping" {
		t.Fatalf("unexpected method consts: %q %q", MethodPlayerOpen, MethodJSONRPCPing)
	}
	if NotificationPlayerOnPlay != "Player.OnPlay" {
		t.Fatalf("unexpected notification const: %q", NotificationPlayerOnPlay)
	}
	if APIMajor != 12 || SchemaVersion != "12.4.0" {
		t.Fatalf("unexpected version: major=%d schema=%q", APIMajor, SchemaVersion)
	}
}

func TestStructRoundTrip(t *testing.T) {
	orig := GlobalTime{Hours: 1, Minutes: 2, Seconds: 3, Milliseconds: 4}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var back GlobalTime
	if err := json.Unmarshal(b, &back); err != nil || back != orig {
		t.Fatalf("round trip failed: %+v %v", back, err)
	}
}

// TestClientCall exercises a generated method against a fake Kodi server.
func TestClientCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch req.Method {
		case "JSONRPC.Ping":
			result = "pong"
		case "Player.GetActivePlayers":
			result = []any{map[string]any{"playerid": 1, "playertype": "internal", "type": "video"}}
		default:
			t.Errorf("unexpected method %q", req.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": result,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx := context.Background()

	if pong, err := c.JSONRPCPing(ctx); err != nil || pong != "pong" {
		t.Fatalf("Ping = %q, %v", pong, err)
	}
	players, err := c.PlayerGetActivePlayers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 1 || players[0].PlayerID != 1 || players[0].Type != PlayerTypeVideo {
		t.Fatalf("unexpected players: %+v", players)
	}
}
