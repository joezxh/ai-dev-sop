package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewOllama(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		BaseURL: baseURL, Model: model,
		HTTP: &http.Client{Timeout: 120 * time.Second},
	}
}

type ollamaReq struct {
	Model    string      `json:"model"`
	Messages []ollamaMsg `json:"messages"`
	Stream   bool        `json:"stream"`
	Format   string      `json:"format,omitempty"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResp struct {
	Message   ollamaMsg `json:"message"`
	EvalCount int       `json:"eval_count"`
}

func (p *OllamaProvider) Complete(ctx context.Context, req Request) (*Response, error) {
	msgs := []ollamaMsg{}
	if req.System != "" {
		msgs = append(msgs, ollamaMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, ollamaMsg{Role: string(m.Role), Content: m.Content})
	}
	body := ollamaReq{Model: p.Model, Messages: msgs, Stream: false}
	if req.Format == "json" {
		body.Format = "json"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.BaseURL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	raw, _ = io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrProviderUnavailable, resp.StatusCode, string(raw))
	}
	var out ollamaResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode: %w body=%s", err, string(raw))
	}
	return &Response{Text: out.Message.Content, Tokens: out.EvalCount}, nil
}

func (p *OllamaProvider) Name() string { return "ollama" }
