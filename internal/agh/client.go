// SPDX-License-Identifier: GPL-3.0-or-later
package agh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Port           int
	User, Password string
	HTTP           *http.Client
}

func New(p int, u, s string) *Client {
	return &Client{p, u, s, &http.Client{Timeout: 4 * time.Second, Transport: &http.Transport{Proxy: nil}}}
}
func (c *Client) Call(method, path string, in, out any) error {
	b, e := json.Marshal(in)
	if e != nil {
		return e
	}
	var body io.Reader
	if in != nil {
		body = bytes.NewReader(b)
	}
	r, e := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d/control%s", c.Port, path), body)
	if e != nil {
		return e
	}
	r.SetBasicAuth(c.User, c.Password)
	r.Header.Set("Content-Type", "application/json")
	resp, e := c.HTTP.Do(r)
	if e != nil {
		return errors.New("AGH API 连接失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("AGH API %s HTTP %d", path, resp.StatusCode)
	}
	if out == nil {
		_, e = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<20))
		return e
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}
func (c *Client) Protection(on bool) error {
	return c.Call("POST", "/protection", map[string]any{"enabled": on, "duration": 0}, nil)
}
func (c *Client) Upstreams(u, b []string, mode string) error {
	return c.Call("POST", "/dns_config", map[string]any{"upstream_dns": u, "bootstrap_dns": b, "fallback_dns": []string{}, "upstream_mode": mode, "upstream_dns_file": ""}, nil)
}
