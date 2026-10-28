package duck

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"patra-world-social/server/internal/games/model"
)

//go:embed rules.v1.json
var rulesAsset []byte

type Rules struct {
	ID              string             `json:"id"`
	Version         int                `json:"version"`
	CountdownMs     int64              `json:"countdownMs"`
	DurationMs      int64              `json:"durationMs"`
	SpawnMs         int64              `json:"spawnMs"`
	LifetimeMs      int64              `json:"lifetimeMs"`
	CooldownMs      int64              `json:"cooldownMs"`
	MaxTargets      int                `json:"maxTargets"`
	MaxRewindMs     int64              `json:"maxRewindMs"`
	MaxCatchUpSteps int64              `json:"maxCatchUpSteps"`
	Radius          float64            `json:"radius"`
	Points          map[string]int64   `json:"points"`
	Speeds          map[string]float64 `json:"speeds"`
}

func DefaultRules() Rules {
	var r Rules
	if err := json.Unmarshal(rulesAsset, &r); err != nil {
		panic(err)
	}
	return r
}
func RulesChecksum() string { s := sha256.Sum256(rulesAsset); return hex.EncodeToString(s[:]) }

type RNG struct{ state uint32 }

func NewRNG(seed int64) *RNG {
	s := uint32(seed)
	if s == 0 {
		s = 1
	}
	return &RNG{s}
}
func (r *RNG) Intn(n int) int {
	s := r.state
	s ^= s << 13
	s ^= s >> 17
	s ^= s << 5
	r.state = s
	return int(s % uint32(n))
}

type Aim struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type target struct {
	ID       int
	Kind     string
	Born     int64
	X, Y, VX float64
	Dead     bool
}
type PublicTarget struct {
	ID        int     `json:"id"`
	Kind      string  `json:"kind"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Radius    float64 `json:"radius"`
	VX        float64 `json:"vx"`
	ExpiresAt int64   `json:"expiresAt"`
}

func spawn(rng model.RNG, id int, born int64, r Rules) target {
	roll := rng.Intn(10)
	kind := "normal"
	if roll == 9 {
		kind = "gold"
	} else if roll >= 6 {
		kind = "fast"
	}
	direction := 1.
	x := 100.
	if rng.Intn(2) == 1 {
		direction = -1
		x = 900
	}
	return target{ID: id, Kind: kind, Born: born, X: x, Y: float64(200 + rng.Intn(550)), VX: direction * r.Speeds[kind]}
}
func (t target) public(at int64, r Rules) PublicTarget {
	return PublicTarget{t.ID, t.Kind, (t.X + t.VX*float64(at-t.Born)/1000) / 1000, t.Y / 1000, r.Radius / 1000, t.VX / 1000, t.Born + r.LifetimeMs}
}
