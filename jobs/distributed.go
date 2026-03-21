package jobs

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dmesha3/elgon/db"
)

// SQLBackendConfig controls the distributed SQL queue backend.
type SQLBackendConfig struct {
	Table        string
	Dialect      string
	NodeID       string
	PollInterval time.Duration
	RetryDelay   time.Duration
}

// SQLBackend is a distributed queue backend backed by a shared SQL table.
type SQLBackend struct {
	adapter db.Adapter
	cfg     SQLBackendConfig
	once    sync.Once
	closed  chan struct{}
	clock   func() time.Time
}

func NewSQLBackend(adapter db.Adapter, cfg SQLBackendConfig) *SQLBackend {
	if cfg.Table == "" {
		cfg.Table = "elgon_jobs"
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 250 * time.Millisecond
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 2 * time.Second
	}
	if cfg.NodeID == "" {
		cfg.NodeID = "node_" + randHex(4)
	}
	return &SQLBackend{adapter: adapter, cfg: cfg, closed: make(chan struct{}), clock: time.Now}
}

func (q *SQLBackend) Enqueue(ctx context.Context, msg Message) error {
	if err := q.ensureTable(ctx); err != nil {
		return err
	}
	id := randHex(12)
	now := q.clock().UnixMilli()
	query := fmt.Sprintf("INSERT INTO %s (id, name, payload_b64, status, attempts, available_at_ms, created_at_ms) VALUES (%s, %s, %s, 'queued', 0, %s, %s)",
		q.cfg.Table, q.ph(1), q.ph(2), q.ph(3), q.ph(4), q.ph(5))
	_, err := q.adapter.ExecContext(ctx, query, id, msg.Name, base64.RawStdEncoding.EncodeToString(msg.Payload), now, now)
	return err
}

func (q *SQLBackend) RunWorker(ctx context.Context, handler Handler) {
	ticker := time.NewTicker(q.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.closed:
			return
		case <-ticker.C:
			jobID, msg, ok, err := q.claimNext(ctx)
			if err != nil || !ok {
				continue
			}
			if err := handler(ctx, msg); err != nil {
				_ = q.retry(ctx, jobID)
				continue
			}
			_ = q.ack(ctx, jobID)
		}
	}
}

func (q *SQLBackend) Close() {
	select {
	case <-q.closed:
		return
	default:
		close(q.closed)
	}
}

func (q *SQLBackend) ensureTable(ctx context.Context) error {
	var err error
	q.once.Do(func() {
		stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	payload_b64 TEXT NOT NULL,
	status TEXT NOT NULL,
	attempts INTEGER NOT NULL,
	available_at_ms BIGINT NOT NULL,
	reserved_by TEXT,
	reserved_at_ms BIGINT,
	created_at_ms BIGINT NOT NULL
)`, q.cfg.Table)
		_, err = q.adapter.ExecContext(ctx, stmt)
	})
	return err
}

func (q *SQLBackend) claimNext(ctx context.Context) (string, Message, bool, error) {
	if err := q.ensureTable(ctx); err != nil {
		return "", Message{}, false, err
	}
	now := q.clock().UnixMilli()
	selectStmt := fmt.Sprintf("SELECT id, name, payload_b64 FROM %s WHERE status='queued' AND available_at_ms <= %s ORDER BY created_at_ms ASC LIMIT 1", q.cfg.Table, q.ph(1))
	rows, err := q.adapter.QueryContext(ctx, selectStmt, now)
	if err != nil {
		return "", Message{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", Message{}, false, rows.Err()
	}
	var id, name, payloadB64 string
	if err := rows.Scan(&id, &name, &payloadB64); err != nil {
		return "", Message{}, false, err
	}

	claimStmt := fmt.Sprintf("UPDATE %s SET status='processing', reserved_by=%s, reserved_at_ms=%s, attempts=attempts+1 WHERE id=%s AND status='queued'", q.cfg.Table, q.ph(1), q.ph(2), q.ph(3))
	res, err := q.adapter.ExecContext(ctx, claimStmt, q.cfg.NodeID, now, id)
	if err != nil {
		return "", Message{}, false, err
	}
	affected, _ := res.RowsAffected()
	if affected != 1 {
		return "", Message{}, false, nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", Message{}, false, err
	}
	return id, Message{Name: name, Payload: payload}, true, nil
}

func (q *SQLBackend) ack(ctx context.Context, id string) error {
	stmt := fmt.Sprintf("DELETE FROM %s WHERE id=%s", q.cfg.Table, q.ph(1))
	_, err := q.adapter.ExecContext(ctx, stmt, id)
	return err
}

func (q *SQLBackend) retry(ctx context.Context, id string) error {
	next := q.clock().Add(q.cfg.RetryDelay).UnixMilli()
	stmt := fmt.Sprintf("UPDATE %s SET status='queued', available_at_ms=%s WHERE id=%s", q.cfg.Table, q.ph(1), q.ph(2))
	_, err := q.adapter.ExecContext(ctx, stmt, next, id)
	return err
}

func (q *SQLBackend) ph(n int) string {
	if strings.EqualFold(q.cfg.Dialect, "postgres") || strings.EqualFold(q.cfg.Dialect, "pg") {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func randHex(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000")))
	}
	return hex.EncodeToString(buf)
}
