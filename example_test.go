package kodi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/ayasechan/kodigo"
	"github.com/ayasechan/kodigo/v13"
)

func ExampleNegotiateClient() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]any{
				"version": map[string]any{"major": 13, "minor": 5, "patch": 0},
			},
		})
	}))
	defer srv.Close()

	_, n, err := kodi.NegotiateClient(context.Background(), srv.URL)
	fmt.Println(err == nil, n.Compatibility, n.Server)
	// Output: true compatible 13.5.0
}

// Example_versionSwitch shows how to pick a versioned client after probing
// an unknown server, reusing the same transport (no reconnect).
// It uses only public API.
func Example_versionSwitch() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result any = map[string]any{
			"version": map[string]any{"major": 13, "minor": 5, "patch": 0},
		}
		if req.Method == "Player.GetActivePlayers" {
			result = []any{
				map[string]any{"playerid": 1, "playertype": "internal", "type": "video"},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": result,
		})
	}))
	defer srv.Close()

	ctx := context.Background()
	base := kodi.NewClient(srv.URL)
	n, err := kodi.Negotiate(ctx, base)
	if err != nil {
		panic(err)
	}
	var players v13.PlayerGetActivePlayersResult
	switch n.Server.Major {
	case v13.APIMajor:
		players, err = v13.Wrap(base.Client).PlayerGetActivePlayers(ctx)
	default:
		panic("unsupported version")
	}
	if err != nil {
		panic(err)
	}
	fmt.Println(n.Compatibility, len(players), players[0].Type)
	// Output: compatible 1 video
}
