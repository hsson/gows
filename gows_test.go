package gows_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/hsson/gows"
)

func TestEcho(t *testing.T) {
	echo := func(w http.ResponseWriter, r *http.Request) {
		ws, err := gows.Upgrade(w, r)
		if err != nil {
			t.Logf("failed to upgrade connection: %v", err)
			return
		}
		for {
			msg := <-ws.Read()
			err := ws.WriteText(msg.Data)
			if err != nil {
				t.Logf("got err when writing to socket: %v", err)
				return
			}
		}
	}

	s := httptest.NewServer(http.HandlerFunc(echo))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	// Send message to server, read response and check to see if it's what we expect.
	for i := 0; i < 10; i++ {
		if err := ws.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
			t.Fatalf("%v", err)
		}
		_, p, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if string(p) != "hello" {
			t.Fatalf("bad message: %s", p)
		}
	}
}
