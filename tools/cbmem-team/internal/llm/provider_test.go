package llm

import (
	"context"
	"strings"
	"testing"
)

func TestFakeProviderCompletes(t *testing.T) {
	p := NewFake(`{"text":"hello"}`)
	resp, err := p.Complete(context.Background(), Request{System: "s", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "hello") {
		t.Fatalf("got %q", resp.Text)
	}
	if resp.Tokens == 0 {
		t.Fatal("tokens not counted")
	}
	if got := p.Name(); got != "fake" {
		t.Fatalf("name=%s", got)
	}
}
