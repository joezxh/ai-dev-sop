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

type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

func NewOpenAI(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		BaseURL: baseURL, APIKey: apiKey, Model: model,
		HTTP: &http.Client{Timeout: 60 * time.Second},
	}
}

type openaiReq struct {
	Model    string         `json:"model"`
	Messages []openaiMsg    `json:"messages"`
	Format   map[string]any `json:"response_format,omitempty"`
}

type openaiMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResp struct {
	Choices []struct {
		Message openaiMsg `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req Request) (*Response, error) {
	msgs := []openaiMsg{}
	if req.System != "" {
		msgs = append(msgs, openaiMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, openaiMsg{Role: string(m.Role), Content: m.Content})
	}
	body := openaiReq{Model: p.Model, Messages: msgs}
	if req.Format == "json" {
		body.Format = map[string]any{"type": "json_object"}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	raw, _ = io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrProviderUnavailable, resp.StatusCode, string(raw))
	}
	var out openaiResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode: %w body=%s", err, string(raw))
	}
	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}
	return &Response{Text: text, Tokens: out.Usage.TotalTokens}, nil
}

func (p *OpenAIProvider) Name() string { return "openai" }
