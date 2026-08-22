package evidence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/evidence"
)

type memoryBlobs map[string][]byte

func (b memoryBlobs) Stage(_ context.Context, key string, body []byte, _ string) error {
	b[key] = append([]byte(nil), body...)
	return nil
}
func (b memoryBlobs) Read(_ context.Context, key string) ([]byte, error) {
	body, ok := b[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return append([]byte(nil), body...), nil
}
func (b memoryBlobs) Promote(_ context.Context, from, to string) error {
	b[to] = b[from]
	delete(b, from)
	return nil
}
func (b memoryBlobs) Delete(_ context.Context, key string) error { delete(b, key); return nil }

type memoryCatalog struct{ object evidence.Object }

func (c *memoryCatalog) Commit(_ context.Context, object evidence.Object) error {
	c.object = object
	return nil
}
func (c *memoryCatalog) Find(_ context.Context, projectID, objectID int64) (evidence.Object, error) {
	if c.object.ProjectID != projectID || c.object.ID != objectID {
		return evidence.Object{}, errors.New("not found")
	}
	return c.object, nil
}

func TestStorePublishesOnlyVerifiedOwnedBytes(t *testing.T) {
	t.Parallel()
	blobs := memoryBlobs{}
	catalog := &memoryCatalog{}
	store, err := evidence.NewStore(blobs, catalog)
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.Put(context.Background(), 1, 2, "application/json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	body, err := store.Get(context.Background(), 2, object.ID)
	if err != nil || string(body) != `{"ok":true}` {
		t.Fatalf("IT-032 unexpected evidence: %q %v", body, err)
	}
	blobs[object.Key] = []byte("corrupt")
	if _, err := store.Get(context.Background(), 2, object.ID); !errors.Is(err, evidence.ErrCorrupt) {
		t.Fatalf("UT-259 got %v", err)
	}
}
