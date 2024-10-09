package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/snowmerak/redis-client-side-cache-note/lib/redis"
	"github.com/snowmerak/redis-client-side-cache-note/lib/redis/goredis"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	hm, err := goredis.NewClient(ctx, redis.NewConfig().SetAddresses([]string{"localhost:6379"}))
	if err != nil {
		panic(err)
	}

	if err := hm.Tracking(ctx); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			slog.Info("context done")
			return
		case <-ticker.C:
			v, err := hm.Get(ctx, "key")
			if err != nil {
				continue
			}
			slog.Info("getting key", "value", v)
		}
	}
}
