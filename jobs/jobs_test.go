package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestInMemoryQueue(t *testing.T) {
	q := NewInMemoryQueue(2)
	defer q.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go q.RunWorker(ctx, func(_ context.Context, msg Message) error {
		if msg.Name != "hello" {
			t.Fatalf("unexpected message: %+v", msg)
		}
		close(done)
		return nil
	})

	if err := q.Enqueue(ctx, Message{Name: "hello"}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not process message")
	}
}

func TestScheduler(t *testing.T) {
	s := NewScheduler()
	var count int32
	if err := s.Add("tick", "@every 10ms", func(context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	s.Start(ctx)

	if atomic.LoadInt32(&count) < 2 {
		t.Fatalf("expected at least 2 runs, got %d", count)
	}
}
