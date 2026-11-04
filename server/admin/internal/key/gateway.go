package key

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogu-x/proxy-access-manager/model"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	config Config
	http   *http.Client
}

func NewProxy(c Config) *Client { return &Client{c, &http.Client{Timeout: 5 * time.Second}} }
func (c *Client) command(ctx context.Context, args ...string) ([]byte, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, c.config.XrayBin, append([]string{"api"}, args...)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Xray API: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
func (c *Client) Present(ctx context.Context, k model.ProxyKey) (bool, error) {
	out, err := c.command(ctx, "inbounduser", "--server="+c.config.XrayAPI, "-tag="+c.config.XrayTag, "-email="+k.ID+"@keymgr.local")
	if err != nil {
		return false, err
	}
	var result struct {
		Users []json.RawMessage `json:"users"`
	}
	if err = json.Unmarshal(out, &result); err != nil {
		return false, err
	}
	return len(result.Users) > 0, nil
}
func (c *Client) Add(ctx context.Context, k model.ProxyKey) error {
	payload := map[string]any{"inbounds": []any{map[string]any{"tag": c.config.XrayTag, "port": c.config.XrayPort, "protocol": "vless", "settings": map[string]any{"clients": []any{map[string]any{"id": k.ID, "email": k.ID + "@keymgr.local", "flow": "xtls-rprx-vision", "level": 0}}, "decryption": "none"}}}}
	f, err := os.CreateTemp("", "keymgr-xray-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = json.NewEncoder(f).Encode(payload); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	out, err := c.command(ctx, "adu", "--server="+c.config.XrayAPI, f.Name())
	if err != nil {
		return err
	}
	if !bytes.Contains(out, []byte("Added 1 user")) {
		return fmt.Errorf("Xray 添加用户未确认: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
func (c *Client) Remove(ctx context.Context, k model.ProxyKey) error {
	out, err := c.command(ctx, "rmu", "--server="+c.config.XrayAPI, "-tag="+c.config.XrayTag, k.ID+"@keymgr.local")
	if err != nil {
		return err
	}
	if !bytes.Contains(out, []byte("Removed 1 user")) {
		return fmt.Errorf("Xray 删除用户未确认: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
func (c *Client) XrayStat(ctx context.Context, k model.ProxyKey, direction string) (*int64, error) {
	name := "user>>>" + k.ID + "@keymgr.local>>>traffic>>>" + direction
	out, err := c.command(ctx, "stats", "--server="+c.config.XrayAPI, "-name="+name)
	if err != nil {
		return nil, err
	}
	var result struct {
		Stat *struct {
			Value int64 `json:"value"`
		} `json:"stat"`
	}
	if err = json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	if result.Stat == nil {
		return nil, nil
	}
	return &result.Stat.Value, nil
}
func (c *Client) request(ctx context.Context, path, method string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.config.HysteriaStats+path, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", c.config.HysteriaSecret)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Hysteria API 状态 %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, 1<<20))
}

type Traffic struct {
	TX int64 `json:"tx"`
	RX int64 `json:"rx"`
}

func (c *Client) HysteriaTraffic(ctx context.Context) (map[string]Traffic, error) {
	out, err := c.request(ctx, "/traffic", "GET", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]Traffic
	err = json.Unmarshal(out, &result)
	return result, err
}
func (c *Client) Kick(ctx context.Context, k model.ProxyKey) error {
	_, err := c.request(ctx, "/kick", "POST", []string{k.ID})
	return err
}
