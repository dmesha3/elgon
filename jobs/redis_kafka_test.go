package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeRedis struct {
	mu     sync.Mutex
	items  []string
	closed bool
}

func (f *fakeRedis) LPush(_ context.Context, _ string, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append([]string{value}, f.items...)
	return nil
}
func (f *fakeRedis) BRPop(ctx context.Context, _ time.Duration, _ string) (string, error) {
	deadline := time.NewTimer(300 * time.Millisecond)
	defer deadline.Stop()
	for {
		f.mu.Lock()
		if len(f.items) > 0 {
			v := f.items[len(f.items)-1]
			f.items = f.items[:len(f.items)-1]
			f.mu.Unlock()
			return v, nil
		}
		f.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", errors.New("timeout")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func (f *fakeRedis) Close() error { f.closed = true; return nil }

type fakeKafkaProducer struct {
	messages [][]byte
}

func (f *fakeKafkaProducer) Publish(_ context.Context, _ string, _ []byte, value []byte) error {
	f.messages = append(f.messages, value)
	return nil
}
func (f *fakeKafkaProducer) Close() error { return nil }

type fakeKafkaConsumer struct {
	mu       sync.Mutex
	messages [][]byte
}

func (f *fakeKafkaConsumer) Poll(ctx context.Context, _ time.Duration) ([]byte, error) {
	for {
		f.mu.Lock()
		if len(f.messages) > 0 {
			v := f.messages[0]
			f.messages = f.messages[1:]
			f.mu.Unlock()
			return v, nil
		}
		f.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func (f *fakeKafkaConsumer) Close() error { return nil }

func TestRedisQueue(t *testing.T) {
	client := &fakeRedis{}
	q := NewRedisQueue(client, RedisQueueConfig{Key: "jobs"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := q.Enqueue(ctx, Message{Name: "job1", Payload: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go q.RunWorker(ctx, func(_ context.Context, m Message) error {
		if m.Name != "job1" {
			t.Fatalf("unexpected msg: %+v", m)
		}
		close(done)
		return nil
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("redis worker timeout")
	}
}

func TestKafkaQueue(t *testing.T) {
	producer := &fakeKafkaProducer{}
	consumer := &fakeKafkaConsumer{}
	q := NewKafkaQueue(producer, consumer, KafkaQueueConfig{Topic: "jobs"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := q.Enqueue(ctx, Message{Name: "job2", Payload: []byte("y")}); err != nil {
		t.Fatal(err)
	}
	if len(producer.messages) != 1 {
		t.Fatalf("expected published message")
	}
	consumer.messages = append(consumer.messages, producer.messages[0])

	done := make(chan struct{})
	go q.RunWorker(ctx, func(_ context.Context, m Message) error {
		if m.Name != "job2" {
			t.Fatalf("unexpected msg: %+v", m)
		}
		close(done)
		return nil
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("kafka worker timeout")
	}
}

func TestKafkaPayloadEncoding(t *testing.T) {
	producer := &fakeKafkaProducer{}
	q := NewKafkaQueue(producer, nil, KafkaQueueConfig{})
	if err := q.Enqueue(context.Background(), Message{Name: "job3", Payload: []byte("z")}); err != nil {
		t.Fatal(err)
	}
	var m Message
	if err := json.Unmarshal(producer.messages[0], &m); err != nil {
		t.Fatal(err)
	}
	if m.Name != "job3" {
		t.Fatalf("unexpected payload: %+v", m)
	}
}
