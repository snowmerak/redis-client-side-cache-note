package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/redis/rueidis"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"localhost:6379"},
	})
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	done := ctx.Done()

	for {
		select {
		case <-done:
			slog.Info("shutting down")
			return
		case <-ticker.C:
			const key = "key"
			resp := client.DoCache(ctx, client.B().Get().Key(key).Cache(), 10*time.Second)
			if resp.Error() != nil {
				slog.Error("failed to get key", "error", resp.Error())
				continue
			}
			i, err := resp.AsInt64()
			if err != nil {
				slog.Error("failed to convert response to int64", "error", err)
				continue
			}
			switch resp.IsCacheHit() {
			case true:
				slog.Info("cache hit", "key", key)
			case false:
				slog.Info("missed key", "key", key)
			}
			slog.Info("got key", "value", i)
		}
	}
}
