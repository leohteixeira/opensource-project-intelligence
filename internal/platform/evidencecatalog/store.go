// Package evidencecatalog persists immutable object ownership in PostgreSQL.
package evidencecatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/evidence"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *database.Pool) (*Store, error) {
	if pool == nil {
		return nil, evidence.ErrInvalid
	}
	return &Store{pool: pool.Unwrap()}, nil
}

func (store *Store) Commit(ctx context.Context, object evidence.Object) error {
	if object.ID <= 0 || object.ProjectID <= 0 || object.Key == "" || object.Size < 0 || object.MediaType == "" {
		return evidence.ErrInvalid
	}
	command, err := store.pool.Exec(ctx, `INSERT INTO object_references
		(id,object_key,sha256,size_bytes,media_type,retention_state,project_id,verified_at)
		VALUES ($1,$2,$3,$4,$5,'retained',$6,now())
		ON CONFLICT (object_key) DO NOTHING`, object.ID, object.Key, object.Digest[:],
		object.Size, object.MediaType, object.ProjectID)
	if err != nil {
		return fmt.Errorf("commit evidence catalog entry: %w", err)
	}
	if command.RowsAffected() == 1 {
		return nil
	}
	existing, err := store.Find(ctx, object.ProjectID, object.ID)
	if err != nil {
		return fmt.Errorf("resolve evidence catalog conflict: %w", err)
	}
	if existing.Key != object.Key || existing.Digest != object.Digest || existing.Size != object.Size ||
		existing.MediaType != object.MediaType {
		return evidence.ErrInvalid
	}
	return nil
}

func (store *Store) Find(ctx context.Context, projectID, objectID int64) (evidence.Object, error) {
	if projectID <= 0 || objectID <= 0 {
		return evidence.Object{}, evidence.ErrInvalid
	}
	var object evidence.Object
	var digest []byte
	err := store.pool.QueryRow(ctx, `SELECT id,project_id,object_key,sha256,size_bytes,media_type
		FROM object_references WHERE id=$1 AND project_id=$2 AND retention_state='retained'
		AND verified_at IS NOT NULL`, objectID, projectID).Scan(&object.ID, &object.ProjectID,
		&object.Key, &digest, &object.Size, &object.MediaType)
	if errors.Is(err, pgx.ErrNoRows) {
		return evidence.Object{}, evidence.ErrInvalid
	}
	if err != nil {
		return evidence.Object{}, fmt.Errorf("read evidence catalog entry: %w", err)
	}
	if len(digest) != len(object.Digest) {
		return evidence.Object{}, evidence.ErrCorrupt
	}
	copy(object.Digest[:], digest)
	return object, nil
}
