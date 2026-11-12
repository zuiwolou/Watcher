package etcdx

import (
	"context"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client wraps etcd for optional runtime config (BE-18).
type Client struct {
	cli *clientv3.Client
}

// Dial from WATCHER_ETCD_ENDPOINTS (comma-separated). Empty → nil client (noop).
func Dial() (*Client, error) {
	eps := os.Getenv("WATCHER_ETCD_ENDPOINTS")
	if eps == "" {
		return nil, nil
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   splitCSV(eps),
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() {
	if c != nil && c.cli != nil {
		_ = c.cli.Close()
	}
}

// GetString reads a key; missing key → "", nil.
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	if c == nil || c.cli == nil {
		return "", nil
	}
	resp, err := c.cli.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

// Well-known keys
const (
	KeyJWTAccessTTL      = "/watcher/jwt/access_ttl"
	KeyHistoryRetainDays = "/watcher/history/retain_days"
	KeyGeofenceCooldown     = "/watcher/geofence/cooldown"
)

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
