package access

import "testing"

// UT-251 and IT-140: cursor authority is signed, snapshot-bound, and stable.
func TestUT251CursorIsBoundToRouteAndFrozenFilters(t *testing.T) {
	t.Parallel()
	codec, err := NewCursorCodec([]byte("task03-cursor-signing-key-is-long-enough"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Encode(Cursor{Route: "projects", Filters: "state=active&q=go", Offset: 24})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(encoded, "projects", "state=active&q=go")
	if err != nil || decoded.Offset != 24 {
		t.Fatalf("valid cursor = %#v, %v", decoded, err)
	}
	reencoded, err := codec.Encode(decoded)
	if err != nil || reencoded != encoded {
		t.Fatalf("stable cursor = %q, %v; want %q", reencoded, err, encoded)
	}
	for _, mismatch := range []struct{ route, filters string }{
		{"repositories", "state=active&q=go"},
		{"projects", "state=archived&q=go"},
	} {
		if _, err := codec.Decode(encoded, mismatch.route, mismatch.filters); err == nil {
			t.Fatalf("cursor was accepted for route=%q filters=%q", mismatch.route, mismatch.filters)
		}
	}
	if _, err := codec.Decode(encoded[:len(encoded)-1]+"x", "projects", "state=active&q=go"); err == nil {
		t.Fatal("tampered cursor was accepted")
	}
}
