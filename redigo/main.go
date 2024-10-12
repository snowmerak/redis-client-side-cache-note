package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := NewRedisClient("localhost:6379")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	go func() {
		if err := client.Tracking(ctx); err != nil {
			slog.Error("failed to track invalidation message", "error", err)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	done := ctx.Done()

	for {
		select {
		case <-done:
			slog.Info("shutting down")
			return
		case <-ticker.C:
			v, err := client.Get("key")
			if err != nil {
				slog.Error("failed to get key", "error", err)
				return
			}
			slog.Info("got key", "value", v)
		}
	}
}
