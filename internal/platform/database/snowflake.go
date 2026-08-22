package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/leohteixeira/opensource-project-intelligence/internal/platform/database/sqlc"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/id"
)

// SnowflakeLeaser coordinates exclusive identifier nodes through PostgreSQL.
type SnowflakeLeaser struct {
	queries *dbgen.Queries
}

// NewSnowflakeLeaser binds generated queries to the shared pool.
func NewSnowflakeLeaser(pool *Pool) *SnowflakeLeaser {
	return &SnowflakeLeaser{queries: dbgen.New(pool.Unwrap())}
}

// Acquire implements id.Leaser. PostgreSQL serializes competing node claims.
func (l *SnowflakeLeaser) Acquire(
	ctx context.Context,
	holder string,
	now time.Time,
	ttl time.Duration,
) (id.Lease, error) {
	row, err := l.queries.AcquireSnowflakeNode(ctx, dbgen.AcquireSnowflakeNodeParams{
		HolderID:       holder,
		LeaseExpiresAt: pgtype.Timestamptz{Time: now.Add(ttl), Valid: true},
		NowAt:          pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return id.Lease{}, fmt.Errorf("database: acquire Snowflake node: %w", err)
	}
	if !row.LeaseExpiresAt.Valid || row.NodeID < 0 {
		return id.Lease{}, fmt.Errorf("database: acquire Snowflake node: %w", id.ErrLeaseLost)
	}

	return id.Lease{
		Node:      uint16(row.NodeID),
		Holder:    row.HolderID,
		ExpiresAt: row.LeaseExpiresAt.Time,
	}, nil
}

// Renew implements id.Leaser. The holder and node must still own an unexpired lease.
func (l *SnowflakeLeaser) Renew(
	ctx context.Context,
	lease id.Lease,
	now time.Time,
	ttl time.Duration,
) (id.Lease, error) {
	row, err := l.queries.RenewSnowflakeNode(ctx, dbgen.RenewSnowflakeNodeParams{
		LeaseExpiresAt: pgtype.Timestamptz{Time: now.Add(ttl), Valid: true},
		NowAt:          pgtype.Timestamptz{Time: now, Valid: true},
		NodeID:         int16(lease.Node),
		HolderID:       lease.Holder,
	})
	if err != nil {
		return id.Lease{}, fmt.Errorf("database: renew Snowflake node: %w", err)
	}
	if !row.LeaseExpiresAt.Valid || row.NodeID < 0 {
		return id.Lease{}, fmt.Errorf("database: renew Snowflake node: %w", id.ErrLeaseLost)
	}

	return id.Lease{
		Node:      uint16(row.NodeID),
		Holder:    row.HolderID,
		ExpiresAt: row.LeaseExpiresAt.Time,
	}, nil
}
