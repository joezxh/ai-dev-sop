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

type MemPalace struct {
	BaseURL string
	HTTP    *http.Client
}

func NewMemPalace(baseURL, _ string) *MemPalace {
	return &MemPalace{BaseURL: baseURL, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

type drawerReq struct {
	Wing    string `json:"wing"`
	Room    string `json:"room"`
	Hall    string `json:"hall"`
	Content string `json:"content"`
}

func (m *MemPalace) AddDrawer(ctx context.Context, wing, room, hall, content string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := json.Marshal(drawerReq{Wing: wing, Room: room, Hall: hall, Content: content})
	if err != nil {
		return fmt.Errorf("marshal drawer: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		m.BaseURL+"/api/drawers", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := m.HTTP.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			return nil
		}
		lastErr = fmt.Errorf("mempalace status=%d body=%s", resp.StatusCode, string(body))
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return lastErr
}

func (m *MemPalace) Status(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.BaseURL+"/healthz", nil)
	if err != nil {
		return "", err
	}
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "down", nil
	}
	return "up", nil
}
