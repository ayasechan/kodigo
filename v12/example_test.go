package v12_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/ayasechan/kodigo/v12"
)

func ExampleClient_JSONRPCPing() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": "pong",
		})
	}))
	defer srv.Close()

	pong, err := v12.NewClient(srv.URL).JSONRPCPing(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(pong)
	// Output: pong
}
