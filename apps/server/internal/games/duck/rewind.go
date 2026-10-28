package duck

import (
	"crypto/rand"
	"encoding/hex"
	"math"
)

type ClockChallenge struct {
	Nonce  string `json:"nonce"`
	SentAt int64  `json:"sentAt"`
}
type clockSample struct {
	rtt, at int64
	offset  float64
}

// RTT includes queueing and client processing. The minimum of four recent
// samples is conservative; asymmetric routes cannot be measured by this exchange.
// Client time validates plausibility only: server RTT chooses all rewind.
type clockEstimator struct {
	epoch      uint64
	pending    ClockChallenge
	samples    []clockSample
	lastClient float64
	hasClient  bool
}

func (c *clockEstimator) bind(epoch uint64) {
	if c.epoch != epoch {
		*c = clockEstimator{epoch: epoch}
	}
}
func (c *clockEstimator) challenge(epoch uint64, at int64) ClockChallenge {
	c.bind(epoch)
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	c.pending = ClockChallenge{hex.EncodeToString(b[:]), at}
	return c.pending
}
func (c *clockEstimator) observe(epoch uint64, nonce string, client float64, at int64) error {
	c.bind(epoch)
	p := c.pending
	if p.Nonce == "" || nonce != p.Nonce {
		return ErrInput
	}
	c.pending = ClockChallenge{}
	rtt := at - p.SentAt
	if !finiteTime(client) || rtt < 0 || rtt > 1000 {
		return ErrInput
	}
	offset := float64(p.SentAt) + float64(rtt)/2 - client
	for _, s := range c.samples {
		if at-s.at < 15000 && math.Abs(offset-s.offset) > 100+float64(rtt) {
			return ErrInput
		}
	}
	c.samples = append(c.samples, clockSample{rtt, at, offset})
	if len(c.samples) > 4 {
		c.samples = append([]clockSample{}, c.samples[len(c.samples)-4:]...)
	}
	return nil
}
func finiteTime(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 9007199254740991
}
func (c *clockEstimator) rewind(epoch uint64, client float64, at int64) (int64, error) {
	c.bind(epoch)
	if !finiteTime(client) || c.hasClient && client <= c.lastClient {
		return 0, ErrInput
	}
	v, ok := c.calibration(at)
	rewind := int64(0)
	if ok {
		rewind = v.RewindMs
		display := client + v.OffsetMs
		expected := float64(at) - float64(v.RTTMs)/2
		if display > float64(at) || math.Abs(display-expected) > 50 {
			return 0, ErrInput
		}
	}
	c.lastClient = client
	c.hasClient = true
	return rewind, nil
}

// ClockCalibration is private to the admitted shooter's command ACK. Offset and
// RTT come from the same eligible sample used to select authoritative rewind.
type ClockCalibration struct {
	OffsetMs   float64 `json:"offsetMs"`
	RTTMs      int64   `json:"rttMs"`
	RewindMs   int64   `json:"rewindMs"`
	MeasuredAt int64   `json:"measuredAt"`
	ExpiresAt  int64   `json:"expiresAt"`
}

func (c *clockEstimator) calibration(at int64) (ClockCalibration, bool) {
	var best *clockSample
	for i := range c.samples {
		s := &c.samples[i]
		if at >= s.at && at-s.at <= 15000 && (best == nil || s.rtt < best.rtt) {
			best = s
		}
	}
	if best == nil {
		return ClockCalibration{}, false
	}
	return ClockCalibration{best.offset, best.rtt, min(int64(150), best.rtt/2), best.at, best.at + 15000}, true
}
