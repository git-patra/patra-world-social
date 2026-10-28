package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"unicode/utf8"
)

func Object(raw []byte) (map[string]any, error) {
	if len(raw) > 1048576 {
		return nil, errors.New("oversized JSON")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var v map[string]any
	if e := d.Decode(&v); e != nil || v == nil {
		return nil, errors.New("invalid object")
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, errors.New("trailing JSON")
	}
	return v, nil
}
func Number(v any, min, max float64) bool {
	n, ok := v.(float64)
	return ok && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= min && n <= max
}
func Integer(v any, min, max float64) bool {
	n, ok := v.(float64)
	return ok && Number(v, min, max) && math.Trunc(n) == n
}
func Text(v any, max int) bool {
	s, ok := v.(string)
	return ok && utf8.RuneCountInString(s) > 0 && utf8.RuneCountInString(s) <= max
}
func Point(v any) bool {
	p, ok := v.(map[string]any)
	return ok && Keys(p, "x", "y") && Number(p["x"], 0, 63) && Number(p["y"], 0, 63)
}

func Keys(v map[string]any, allowed ...string) bool {
	for k := range v {
		found := false
		for _, a := range allowed {
			if k == a {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
