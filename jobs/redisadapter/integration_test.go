//go:build adapters && integration
// +build adapters,integration

package redisadapter

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisAdapterIntegration(t *testing.T) {
	addr := os.Getenv("ELGON_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	client, err := New(Config{Addr: addr})
	if err != nil {
		t.Skipf("redis unavailable at %s: %v", addr, err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := "elgon:itest:queue"
	if err := client.LPush(ctx, key, "payload-1"); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	got, err := client.BRPop(ctx, time.Second, key)
	if err != nil {
		t.Fatalf("brpop: %v", err)
	}
	if got != "payload-1" {
		t.Fatalf("unexpected payload: %q", got)
	}
}
