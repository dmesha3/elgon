package jobs

import (
	"context"
	"errors"
	"sync"
)

var ErrQueueClosed = errors.New("jobs: queue closed")

// Message is a queued job payload.
type Message struct {
	Name    string
	Payload []byte
}

// Handler handles queued messages.
type Handler func(context.Context, Message) error

// Queue defines enqueue and worker behavior.
type Queue interface {
	Enqueue(ctx context.Context, msg Message) error
	RunWorker(ctx context.Context, handler Handler)
	Close()
}

// InMemoryQueue is a lightweight queue for dev and tests.
type InMemoryQueue struct {
	ch     chan Message
	closed bool
	mu     sync.RWMutex
}

func NewInMemoryQueue(buffer int) *InMemoryQueue {
	if buffer < 1 {
		buffer = 1
	}
	return &InMemoryQueue{ch: make(chan Message, buffer)}
}

func (q *InMemoryQueue) Enqueue(ctx context.Context, msg Message) error {
	q.mu.RLock()
	closed := q.closed
	q.mu.RUnlock()
	if closed {
		return ErrQueueClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- msg:
		return nil
	}
}

func (q *InMemoryQueue) RunWorker(ctx context.Context, handler Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-q.ch:
			if !ok {
				return
			}
			_ = handler(ctx, msg)
		}
	}
}

func (q *InMemoryQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
	close(q.ch)
}
