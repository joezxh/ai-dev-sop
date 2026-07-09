package llm

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	System     string
	Messages   []Message
	Format     string
	JSONSchema string
	Timeout    time.Duration
}

type Response struct {
	Text   string
	Tokens int
}

type Provider interface {
	Complete(ctx context.Context, req Request) (*Response, error)
	Name() string
}

var ErrProviderUnavailable = errors.New("llm provider unavailable")

func ReadJSON[T any](r *Response) (T, error) {
	var v T
	if err := json.Unmarshal([]byte(r.Text), &v); err != nil {
		return v, err
	}
	return v, nil
}
