// Package trackgeom shares the immutable track asset and containment without world dependencies.
package trackgeom

import _ "embed"

//go:embed track.v1.json
var Asset []byte

type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func Driveable(o, i Rect, x, y, r float64) bool {
	return x >= float64(o.X)+r && y >= float64(o.Y)+r && x <= float64(o.X+o.Width)-r && y <= float64(o.Y+o.Height)-r && !(x > float64(i.X)-r && x < float64(i.X+i.Width)+r && y > float64(i.Y)-r && y < float64(i.Y+i.Height)+r)
}
