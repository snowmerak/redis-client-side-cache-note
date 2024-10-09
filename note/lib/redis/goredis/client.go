package goredis

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/snowmerak/redis-client-side-cache-note/lib/redis"
)

type Cache struct {
	m sync.Map
}

func (c *Cache) Get(key string) (goredis.Cmder, bool) {
	if v, ok := c.m.Load(key); ok {
		return v.(goredis.Cmder), true
	}

	return nil, false
}

func (c *Cache) Set(key string, cmd goredis.Cmder) {
	c.m.Store(key, cmd)
}

func (c *Cache) ReplaceOnNil(key string, cmd goredis.Cmder) {
	c.m.CompareAndSwap(key, nil, cmd)
}

func (c *Cache) Delete(key string) {
	c.m.Delete(key)
}

func (c *Cache) Clear() {
	c.m = sync.Map{}
}

type Client struct {
	client *goredis.Client
	cache  *Cache
}

func NewClient(ctx context.Context, cfg *redis.Config) (*Client, error) {
	addr := cfg.Addresses()[rand.IntN(len(cfg.Addresses()))]
	conn := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Username: cfg.Username(),
		Password: cfg.Password(),
		PoolSize: 0,
	})

	context.AfterFunc(ctx, func() {
		conn.Close()
	})

	return &Client{
		client: conn,
		cache:  &Cache{},
	}, nil
}

// This is not working as expected
// Because the client use different connection to client tracking and subscribe
// We cannot specify the client id to the subscribe connection
func (c *Client) Tracking(ctx context.Context) error {
	clientId, err := c.client.ClientID(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}

	// Make new connection to subscribe in Subscribe method
	// The connection has different client id
	sub := c.client.Subscribe(ctx, "__redis__:invalidate")

	if err := c.client.Do(ctx, "CLIENT", "TRACKING", "ON", "REDIRECT", clientId).Err(); err != nil {
		return fmt.Errorf("failed to enable tracking: %w", err)
	}

	go func() {
		defer sub.Close()
		done := ctx.Done()
		ch := sub.Channel()

		const maxBackOff = 10 * time.Second
		backoff := 100 * time.Millisecond

		for {
			select {
			case msg := <-ch:
				slog.Info("received message", "channel", msg.Channel, "payload", msg.Payload, "playload_slice", msg.PayloadSlice)
				c.cache.Delete(msg.Payload)
				for _, payload := range msg.PayloadSlice {
					c.cache.Delete(payload)
				}
				backoff = 100 * time.Millisecond
			case <-done:
				slog.Info("context done")
				return
			}

			backoff <<= 1
			if backoff > maxBackOff {
				backoff = maxBackOff
			}
			slog.Error("failed to receive message", "backoff", backoff, "error", err)

			time.Sleep(backoff)

			sub.Close()
			sub = c.client.Subscribe(ctx, "__redis__:invalidate")
		}
	}()

	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	cached, ok := c.cache.Get(key)
	if ok {
		slog.Info("cache hit", "key", key)
		switch v := cached.(type) {
		case *goredis.StringCmd:
			return v.Val(), nil
		}
	}
	slog.Info("cache miss", "key", key)

	response := c.client.Get(ctx, key)
	if err := response.Err(); err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	c.cache.Set(key, response)

	return response.Val(), nil
}

func (c *Client) Set(ctx context.Context, key, value string) error {
	response := c.client.Set(ctx, key, value, 0)
	if err := response.Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	c.cache.Set(key, response)

	return nil
}
