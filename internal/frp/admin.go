package frp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AdminClient struct {
	BaseURL  string
	User     string
	Password string
	Client   *http.Client
}

func (c AdminClient) Health(ctx context.Context) error {
	if c.BaseURL == "" {
		return fmt.Errorf("frpc admin base url is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/api/status", nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("frpc admin status returned %s", resp.Status)
	}
	return nil
}

func (c AdminClient) PutStoreConfig(ctx context.Context, endpoint string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(c.BaseURL, "/")+endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("frpc admin store returned %s", resp.Status)
	}
	return nil
}

func (c AdminClient) auth(req *http.Request) {
	if c.User != "" || c.Password != "" {
		req.SetBasicAuth(c.User, c.Password)
	}
}

func (c AdminClient) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 3 * time.Second}
}
