package v13_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/ayasechan/kodigo/v13"
)

// fakeKodi serves canned JSON-RPC results for documentation examples.
func fakeKodi(result any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": result,
		})
	}))
}

func ExampleClient_PlayerGetActivePlayers() {
	srv := fakeKodi([]any{
		map[string]any{"playerid": 1, "playertype": "internal", "type": "video"},
	})
	defer srv.Close()

	players, err := v13.NewClient(srv.URL).PlayerGetActivePlayers(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(len(players), players[0].PlayerID, players[0].Type)
	// Output: 1 1 video
}

func ExampleClient_VideoLibraryGetMovies() {
	srv := fakeKodi(map[string]any{
		"limits": map[string]any{"start": 0, "end": 1, "total": 1},
		"movies": []any{
			map[string]any{"movieid": 42, "label": "Example", "year": 2024},
		},
	})
	defer srv.Close()

	res, err := v13.NewClient(srv.URL).VideoLibraryGetMovies(
		context.Background(),
		v13.VideoLibraryGetMoviesParams{
			Properties: v13.VideoFieldsMovie{"title", "year"},
		},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Movies[0].MovieID, *res.Movies[0].Year)
	// Output: 42 2024
}
