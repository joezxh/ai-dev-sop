package llm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestMemPalaceAddDrawer(t *testing.T) {
	var hit struct {
		sync.Mutex
		body string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Lock()
		defer hit.Unlock()
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		hit.body = string(b)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	mp := NewMemPalace(srv.URL, "")
	err := mp.AddDrawer(nil, "wing-x", "room-y", "hall-z", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	hit.Lock()
	defer hit.Unlock()
	if !strings.Contains(hit.body, "wing-x") {
		t.Fatalf("body missing wing: %s", hit.body)
	}
}
