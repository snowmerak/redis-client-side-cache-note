package rueidis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"

	"github.com/snowmerak/redis-client-side-cache-note/lib/redis"
)

const (
	// DefaultTimeout is the default timeout for the redis client.
	DefaultTimeout = 5 * time.Second
)

type Client struct {
	client rueidis.Client
}

func New(cfg *redis.Config) (*Client, error) {
	conn, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:       cfg.Addresses(),
		Username:          cfg.Username(),
		Password:          cfg.Password(),
		CacheSizeEachConn: 256 << 20, // 256MB
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	return &Client{
		client: conn,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	// v, err := c.client.Do(ctx, c.client.B().Get().Key(key).Build()).ToString()
	v, err := c.client.DoCache(ctx, c.client.B().Get().Key(key).Cache(), DefaultTimeout).ToString()
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return v, nil
}

func (c *Client) Set(ctx context.Context, key, value string) error {
	if err := c.client.Do(ctx, c.client.B().Set().Key(key).Value(value).Build()).Error(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}
