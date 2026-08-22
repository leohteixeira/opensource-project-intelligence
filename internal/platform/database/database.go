// Package database owns the PostgreSQL connection pool.
//
// Data access uses explicit reviewed SQL and generated sqlc adapters. There is
// no ORM: the SQL that produces a metric stays readable and auditable.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

// Pool wraps the pgx connection pool so that callers depend on this package
// rather than on pgx directly.
type Pool struct {
	pool *pgxpool.Pool
}

// Open creates the pool and verifies that the database answers.
//
// The connection URI is never included in an error message, because it carries
// credentials.
func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: cannot parse the connection configuration: %w", redact(err))
	}
	poolConfig.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		if err := pgxvector.RegisterTypes(ctx, connection); err != nil {
			return fmt.Errorf("register pgvector types: %w", err)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("database: cannot create the connection pool: %w", redact(err))
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: the server did not answer: %w", redact(err))
	}

	return &Pool{pool: pool}, nil
}

// Ping reports whether the database currently answers.
func (p *Pool) Ping(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database: ping failed: %w", redact(err))
	}
	return nil
}

// Close releases every connection.
func (p *Pool) Close() {
	p.pool.Close()
}

// Unwrap exposes the underlying pool to the adapters of this layer. Business
// packages receive narrower interfaces instead.
func (p *Pool) Unwrap() *pgxpool.Pool {
	return p.pool
}
