package evidence

import (
	"context"
	"fmt"
)

// BlobStore is the narrow S3-compatible boundary. Implementations must keep a
// staged key private until Promote succeeds and Delete must be idempotent.
type BlobStore interface {
	Stage(context.Context, string, []byte, string) error
	Read(context.Context, string) ([]byte, error)
	Promote(context.Context, string, string) error
	Delete(context.Context, string) error
}

// Catalog is the PostgreSQL ownership boundary. Find returns only committed,
// retained objects; staged or purging objects are never evidence.
type Catalog interface {
	Commit(context.Context, Object) error
	Find(context.Context, int64, int64) (Object, error)
}

type Store struct {
	blobs   BlobStore
	catalog Catalog
}

func NewStore(blobs BlobStore, catalog Catalog) (*Store, error) {
	if blobs == nil || catalog == nil {
		return nil, ErrInvalid
	}
	return &Store{blobs: blobs, catalog: catalog}, nil
}

// Put uses a private staged key, verifies the exact stored bytes, promotes the
// immutable content-addressed object, and only then commits visible ownership.
// A metadata failure removes the promoted orphan best-effort.
func (s *Store) Put(
	ctx context.Context,
	id, projectID int64,
	mediaType string,
	body []byte,
) (Object, error) {
	object, err := NewObject(id, projectID, mediaType, body)
	if err != nil {
		return Object{}, err
	}
	stagedKey := object.Key + fmt.Sprintf(".staged-%d", id)
	if err := s.blobs.Stage(ctx, stagedKey, body, mediaType); err != nil {
		return Object{}, fmt.Errorf("stage evidence: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = s.blobs.Delete(context.WithoutCancel(ctx), stagedKey)
		}
	}()
	staged, err := s.blobs.Read(ctx, stagedKey)
	if err != nil {
		return Object{}, fmt.Errorf("read staged evidence: %w", err)
	}
	if err := object.Verify(staged); err != nil {
		return Object{}, err
	}
	if err := s.blobs.Promote(ctx, stagedKey, object.Key); err != nil {
		return Object{}, fmt.Errorf("promote evidence: %w", err)
	}
	if err := s.catalog.Commit(ctx, object); err != nil {
		_ = s.blobs.Delete(context.WithoutCancel(ctx), object.Key)
		return Object{}, fmt.Errorf("commit evidence ownership: %w", err)
	}
	cleanup = false
	return object, nil
}

func (s *Store) Get(ctx context.Context, projectID, objectID int64) ([]byte, error) {
	object, err := s.catalog.Find(ctx, projectID, objectID)
	if err != nil {
		return nil, err
	}
	if object.ProjectID != projectID {
		return nil, ErrInvalid
	}
	body, err := s.blobs.Read(ctx, object.Key)
	if err != nil {
		return nil, fmt.Errorf("read evidence: %w", err)
	}
	if err := object.Verify(body); err != nil {
		return nil, err
	}
	return body, nil
}
