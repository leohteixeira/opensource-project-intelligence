// Package recovery verifies canonical object references after backup and restore.
package recovery

import "sort"

// Object records one content-addressed object owned by PostgreSQL.
type Object struct {
	Key    string
	SHA256 string
}

// Result names every unresolved canonical reference and unexpected object.
type Result struct {
	Missing    []string
	Mismatched []string
	Orphaned   []string
}

// Clean reports whether PostgreSQL references and restored objects agree.
func (r Result) Clean() bool {
	return len(r.Missing) == 0 && len(r.Mismatched) == 0 && len(r.Orphaned) == 0
}

// Reconcile compares the PostgreSQL manifest with digests calculated from restored bytes.
func Reconcile(expected []Object, actual map[string]string) Result {
	result := Result{}
	referenced := make(map[string]struct{}, len(expected))
	for _, object := range expected {
		referenced[object.Key] = struct{}{}
		digest, exists := actual[object.Key]
		switch {
		case !exists:
			result.Missing = append(result.Missing, object.Key)
		case digest != object.SHA256:
			result.Mismatched = append(result.Mismatched, object.Key)
		}
	}
	for key := range actual {
		if _, exists := referenced[key]; !exists {
			result.Orphaned = append(result.Orphaned, key)
		}
	}
	sort.Strings(result.Missing)
	sort.Strings(result.Mismatched)
	sort.Strings(result.Orphaned)
	return result
}
