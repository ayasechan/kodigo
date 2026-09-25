package v13

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
}

func TestStructRoundTrip(t *testing.T) {
	// Global.Time: all required ints.
	orig := GlobalTime{Hours: 1, Minutes: 2, Seconds: 3, Milliseconds: 4}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"hours":1,"milliseconds":4,"minutes":2,"seconds":3}` {
		t.Fatalf("unexpected GlobalTime JSON: %s", b)
	}
	var back GlobalTime
	if err := json.Unmarshal(b, &back); err != nil || back != orig {
		t.Fatalf("round trip failed: %+v %v", back, err)
	}
}

func TestExtendsEmbed(t *testing.T) {
	// Audio.Details.Song extends Audio.Details.Media (... -> Item.Details.Base{Label}).
	var s AudioDetailsSong
	s.Label = "Song"
	s.SongID = 7
	s.Artist = []string{"a"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["label"] != "Song" || m["songid"] != float64(7) {
		t.Fatalf("embedded/own fields missing: %s", b)
	}
}

func TestOptionalOmitempty(t *testing.T) {
	p := PlayerOpenParams{} // item/options optional
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{}` {
		t.Fatalf("expected empty params object, got %s", b)
	}
	repeat := PlayerRepeatAll
	p.Options = &PlayerOpenParamsOptions{Repeat: &repeat}
	b, _ = json.Marshal(p)
	if !strings.Contains(string(b), `"repeat":"all"`) {
		t.Fatalf("expected repeat in JSON, got %s", b)
	}
}

// TestClientCall spins up a fake Kodi server and exercises generated methods.
func TestClientCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
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

	pong, err := c.JSONRPCPing(ctx)
	if err != nil || pong != "pong" {
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
