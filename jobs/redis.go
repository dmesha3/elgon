package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrRedisClientNil = errors.New("jobs: redis client is nil")

// RedisClient is the minimal client contract for Redis queue operations.
type RedisClient interface {
	LPush(ctx context.Context, key string, value string) error
	BRPop(ctx context.Context, timeout time.Duration, key string) (string, error)
	Close() error
}

// RedisQueueConfig configures a Redis-backed distributed queue.
type RedisQueueConfig struct {
	Key         string
	PopTimeout  time.Duration
	PollBackoff time.Duration
}

// RedisQueue stores messages in Redis list with blocking pop workers.
type RedisQueue struct {
	client RedisClient
	cfg    RedisQueueConfig
}

func NewRedisQueue(client RedisClient, cfg RedisQueueConfig) *RedisQueue {
	if cfg.Key == "" {
		cfg.Key = "elgon:jobs"
	}
	if cfg.PopTimeout <= 0 {
		cfg.PopTimeout = time.Second
	}
	if cfg.PollBackoff <= 0 {
		cfg.PollBackoff = 100 * time.Millisecond
	}
	return &RedisQueue{client: client, cfg: cfg}
}

func (q *RedisQueue) Enqueue(ctx context.Context, msg Message) error {
	if q.client == nil {
		return ErrRedisClientNil
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.cfg.Key, string(b))
}

func (q *RedisQueue) RunWorker(ctx context.Context, handler Handler) {
	if q.client == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		raw, err := q.client.BRPop(ctx, q.cfg.PopTimeout, q.cfg.Key)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(q.cfg.PollBackoff):
				continue
			}
		}
		var msg Message
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			continue
		}
		_ = handler(ctx, msg)
	}
}

func (q *RedisQueue) Close() {
	if q.client != nil {
		_ = q.client.Close()
	}
}
