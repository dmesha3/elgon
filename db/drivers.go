package db

import (
	"fmt"
	"net/url"
)

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c PostgresConfig) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		url.PathEscape(c.DBName),
		url.QueryEscape(sslMode),
	)
}

type MySQLConfig struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
	Params   map[string]string
}

func (c MySQLConfig) DSN() string {
	values := url.Values{}
	for k, v := range c.Params {
		values.Set(k, v)
	}
	query := values.Encode()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
	)
	if query != "" {
		dsn += "?" + query
	}
	return dsn
}

type SQLiteConfig struct {
	Path string
}

func (c SQLiteConfig) DSN() string {
	if c.Path == "" {
		return "file::memory:?cache=shared"
	}
	return c.Path
}

func OpenPostgres(cfg PostgresConfig) (*SQLAdapter, error) {
	return Open("pgx", cfg.DSN())
}

func OpenMySQL(cfg MySQLConfig) (*SQLAdapter, error) {
	return Open("mysql", cfg.DSN())
}

func OpenSQLite(cfg SQLiteConfig) (*SQLAdapter, error) {
	return Open("sqlite", cfg.DSN())
}
