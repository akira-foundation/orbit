package mailpit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) List(ctx context.Context, start, limit int) (ListResult, error) {
	q := "start=" + strconv.Itoa(start) + "&limit=" + strconv.Itoa(limit)
	var out ListResult
	err := c.do(ctx, http.MethodGet, "/api/v1/messages?"+q, nil, &out)
	return out, err
}

func (c *Client) Get(ctx context.Context, id string) (Message, error) {
	var out Message
	err := c.do(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id), nil, &out)
	return out, err
}

func (c *Client) SetRead(ctx context.Context, ids []string, read bool) error {
	return c.do(ctx, http.MethodPut, "/api/v1/messages", map[string]any{"Read": read, "IDs": ids}, nil)
}

func (c *Client) Delete(ctx context.Context, ids []string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/messages", map[string]any{"IDs": ids}, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailpit: %s %s: %s: %s", method, path, resp.Status, string(msg))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
