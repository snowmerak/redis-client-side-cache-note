package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/gomodule/redigo/redis"
)

type RedisClient struct {
	conn  redis.Conn
	cache *ristretto.Cache[string, any]
	addr  string
}

func NewRedisClient(addr string) (*RedisClient, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache: %w", err)
	}

	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		conn:  conn,
		cache: cache,
		addr:  addr,
	}, nil
}

func (r *RedisClient) Close() error {
	err := r.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	return nil
}

func (r *RedisClient) Tracking(ctx context.Context) error {
	psc, err := redis.Dial("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	clientId, err := redis.Int64(psc.Do("CLIENT", "ID"))
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}
	slog.Info("client id", "id", clientId)

	subscriptionResult, err := redis.String(r.conn.Do("CLIENT", "TRACKING", "ON", "REDIRECT", clientId))
	if err != nil {
		return fmt.Errorf("failed to enable tracking: %w", err)
	}
	slog.Info("subscription result", "result", subscriptionResult)

	if err := psc.Send("SUBSCRIBE", "__redis__:invalidate"); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	psc.Flush()

	for {
		msg, err := psc.Receive()
		if err != nil {
			return fmt.Errorf("failed to receive message: %w", err)
		}

		switch msg := msg.(type) {
		case redis.Message:
			slog.Info("received message", "channel", msg.Channel, "data", msg.Data)
			key := string(msg.Data)
			r.cache.Del(key)
		case redis.Subscription:
			slog.Info("subscription", "kind", msg.Kind, "channel", msg.Channel, "count", msg.Count)
		case error:
			return fmt.Errorf("error: %w", msg)
		case []interface{}:
			if len(msg) != 3 || string(msg[0].([]byte)) != "message" || string(msg[1].([]byte)) != "__redis__:invalidate" {
				slog.Warn("unexpected message", "message", msg)
				continue
			}

			contents := msg[2].([]interface{})
			keys := make([]string, len(contents))
			for i, key := range contents {
				keys[i] = string(key.([]byte))
				r.cache.Del(keys[i])
			}
			slog.Info("received invalidation message", "keys", keys)
		default:
			slog.Warn("unexpected message", "type", fmt.Sprintf("%T", msg))
		}
	}
}

func (r *RedisClient) Get(key string) (any, error) {
	val, found := r.cache.Get(key)
	if found {
		switch v := val.(type) {
		case int64:
			slog.Info("cache hit", "key", key)
			return v, nil
		default:
			slog.Warn("unexpected type", "type", fmt.Sprintf("%T", v))
		}
	}
	slog.Info("cache miss", "key", key)

	val, err := redis.Int64(r.conn.Do("GET", key))
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	r.cache.SetWithTTL(key, val, 1, 10*time.Second)
	return val, nil
}
