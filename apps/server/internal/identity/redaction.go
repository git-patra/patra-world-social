// Package identity redacts public copies without mutating immutable game payloads.
package identity

import (
	"encoding/json"
	"sync"
)

type Redactions struct {
	mu    sync.RWMutex
	users map[string]bool
}

func (r *Redactions) Add(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.users == nil {
		r.users = map[string]bool{}
	}
	r.users[id] = true
}
func (r *Redactions) JSON(raw []byte) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.users) == 0 {
		return raw, nil
	}
	var v any
	if e := json.Unmarshal(raw, &v); e != nil {
		return nil, e
	}
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, n := range x {
				visit(n)
			}
		case map[string]any:
			// Only documented identity shapes, never arbitrary chat text or object IDs.
			if id, ok := x["userId"].(string); ok && r.users[id] {
				x["userId"] = ""
				for _, k := range []string{"displayName", "username"} {
					if _, ok := x[k]; ok {
						x[k] = "Akun dihapus"
					}
				}
				for _, k := range []string{"avatarId", "avatar"} {
					if _, ok := x[k]; ok {
						x[k] = "pumpkin"
					}
				}
			}
			if id, ok := x["id"].(string); ok && r.users[id] && x["status"] != nil && x["x"] != nil && x["y"] != nil {
				if _, ok := x["name"]; ok {
					x["name"] = "Akun dihapus"
				}
			}
			for _, n := range x {
				visit(n)
			}
		}
	}
	visit(v)
	return json.Marshal(v)
}
