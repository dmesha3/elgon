//go:build adapters
// +build adapters

package redisadapter

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config configures the Redis adapter client.
type Config struct {
	Addr     string
	Addrs    []string
	Username string
	Password string
	DB       int
}

// Client adapts go-redis to jobs.RedisClient.
type Client struct {
	client redis.UniversalClient
}

func New(cfg Config) (*Client, error) {
	addrs := cfg.Addrs
	if len(addrs) == 0 {
		if cfg.Addr == "" {
			cfg.Addr = "127.0.0.1:6379"
		}
		addrs = []string{cfg.Addr}
	}
	uc := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addrs,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := uc.Ping(ctx).Err(); err != nil {
		_ = uc.Close()
		return nil, err
	}
	return &Client{client: uc}, nil
}

func NewFromUniversal(client redis.UniversalClient) (*Client, error) {
	if client == nil {
		return nil, errors.New("redisadapter: client is nil")
	}
	return &Client{client: client}, nil
}

func (c *Client) LPush(ctx context.Context, key string, value string) error {
	return c.client.LPush(ctx, key, value).Err()
}

func (c *Client) BRPop(ctx context.Context, timeout time.Duration, key string) (string, error) {
	vals, err := c.client.BRPop(ctx, timeout, key).Result()
	if err != nil {
		return "", err
	}
	if len(vals) < 2 {
		return "", redis.Nil
	}
	return vals[1], nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
