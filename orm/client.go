package orm

import "github.com/dmesha3/elgon/db"

// Config controls optional ORM runtime behavior.
type Config struct {
	Dialect string
}

// Client is the typed ORM entrypoint.
type Client struct {
	db  db.Adapter
	cfg Config
}

// New creates a client with default config.
func New(adapter db.Adapter) *Client {
	return NewWithConfig(adapter, Config{})
}

// NewWithConfig creates a client with explicit options.
func NewWithConfig(adapter db.Adapter, cfg Config) *Client {
	return &Client{db: adapter, cfg: cfg}
}

// SQL returns the underlying raw SQL adapter.
func (c *Client) SQL() db.Adapter {
	if c == nil {
		return nil
	}
	return c.db
}

// Table returns a generic table repository.
func (c *Client) Table(name string) *Table {
	if c == nil || c.db == nil {
		return nil
	}
	return &Table{
		db:      c.db,
		dialect: c.cfg.Dialect,
		name:    name,
	}
}
