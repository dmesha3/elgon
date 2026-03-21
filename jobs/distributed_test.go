package jobs

import (
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmesha3/elgon/db"
)

type memResult struct{ affected int64 }

func (r memResult) LastInsertId() (int64, error) { return 0, nil }
func (r memResult) RowsAffected() (int64, error) { return r.affected, nil }

type memRows struct {
	vals [][]any
	i    int
}

func (r *memRows) Close() error { return nil }
func (r *memRows) Err() error   { return nil }
func (r *memRows) Next() bool {
	r.i++
	return r.i <= len(r.vals)
}
func (r *memRows) Scan(dest ...any) error {
	row := r.vals[r.i-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = row[i].(string)
		}
	}
	return nil
}

type memJob struct {
	id         string
	name       string
	payloadB64 string
	status     string
	available  int64
	created    int64
}

type memAdapter struct {
	mu   sync.Mutex
	jobs []memJob
}

func (m *memAdapter) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	upper := strings.ToUpper(strings.TrimSpace(query))
	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"):
		return memResult{affected: 0}, nil
	case strings.HasPrefix(upper, "INSERT INTO"):
		m.jobs = append(m.jobs, memJob{
			id:         args[0].(string),
			name:       args[1].(string),
			payloadB64: args[2].(string),
			status:     "queued",
			available:  args[3].(int64),
			created:    args[4].(int64),
		})
		return memResult{affected: 1}, nil
	case strings.Contains(upper, "SET STATUS='PROCESSING'"):
		id := args[2].(string)
		for i := range m.jobs {
			if m.jobs[i].id == id && m.jobs[i].status == "queued" {
				m.jobs[i].status = "processing"
				return memResult{affected: 1}, nil
			}
		}
		return memResult{affected: 0}, nil
	case strings.HasPrefix(upper, "DELETE FROM"):
		id := args[0].(string)
		for i := range m.jobs {
			if m.jobs[i].id == id {
				m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
				return memResult{affected: 1}, nil
			}
		}
		return memResult{affected: 0}, nil
	case strings.Contains(upper, "SET STATUS='QUEUED'"):
		id := args[1].(string)
		for i := range m.jobs {
			if m.jobs[i].id == id {
				m.jobs[i].status = "queued"
				m.jobs[i].available = args[0].(int64)
				return memResult{affected: 1}, nil
			}
		}
		return memResult{affected: 0}, nil
	default:
		return memResult{}, nil
	}
}
func (m *memAdapter) QueryContext(_ context.Context, query string, args ...any) (db.Rows, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := args[0].(int64)
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT ID, NAME, PAYLOAD_B64") {
		return &memRows{}, nil
	}
	bestIdx := -1
	for i := range m.jobs {
		if m.jobs[i].status == "queued" && m.jobs[i].available <= now {
			if bestIdx == -1 || m.jobs[i].created < m.jobs[bestIdx].created {
				bestIdx = i
			}
		}
	}
	if bestIdx == -1 {
		return &memRows{vals: nil}, nil
	}
	j := m.jobs[bestIdx]
	return &memRows{vals: [][]any{{j.id, j.name, j.payloadB64}}}, nil
}
func (m *memAdapter) BeginTx(context.Context, *sql.TxOptions) (db.Tx, error) { return nil, nil }
func (m *memAdapter) PingContext(context.Context) error                      { return nil }
func (m *memAdapter) Close() error                                           { return nil }

func TestSQLBackendEnqueueAndConsume(t *testing.T) {
	adapter := &memAdapter{}
	backend := NewSQLBackend(adapter, SQLBackendConfig{PollInterval: 5 * time.Millisecond})
	defer backend.Close()

	ctx := context.Background()
	if err := backend.Enqueue(ctx, Message{Name: "email", Payload: []byte("hello")}); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go backend.RunWorker(ctx, func(_ context.Context, msg Message) error {
		if msg.Name != "email" || string(msg.Payload) != "hello" {
			t.Fatalf("unexpected message: %+v", msg)
		}
		close(done)
		return nil
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not process queued message")
	}
}

func TestSQLBackendRetry(t *testing.T) {
	adapter := &memAdapter{}
	backend := NewSQLBackend(adapter, SQLBackendConfig{PollInterval: 5 * time.Millisecond, RetryDelay: 20 * time.Millisecond})
	defer backend.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := backend.Enqueue(ctx, Message{Name: "retry", Payload: []byte("x")}); err != nil {
		t.Fatal(err)
	}

	count := 0
	backend.RunWorker(ctx, func(_ context.Context, msg Message) error {
		count++
		if msg.Name != "retry" {
			t.Fatalf("unexpected msg %s", msg.Name)
		}
		if count == 1 {
			return context.DeadlineExceeded
		}
		return nil
	})

	if count < 2 {
		t.Fatalf("expected retry attempt, got %d", count)
	}

	// internal consistency check: encoded payload should decode.
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	for _, j := range adapter.jobs {
		if _, err := base64.RawStdEncoding.DecodeString(j.payloadB64); err != nil {
			t.Fatalf("invalid payload encoding: %v", err)
		}
	}
}
