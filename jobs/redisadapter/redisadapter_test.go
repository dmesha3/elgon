//go:build adapters
// +build adapters

package redisadapter

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestNewFromUniversalNil(t *testing.T) {
	_, err := NewFromUniversal(nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestNewFromUniversal(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	adapter, err := NewFromUniversal(client)
	if err != nil {
		t.Fatal(err)
	}
	if adapter == nil {
		t.Fatal("expected adapter")
	}
}
