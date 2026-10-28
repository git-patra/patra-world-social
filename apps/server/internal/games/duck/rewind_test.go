package duck

import (
	"math"
	"testing"
	"time"
)

func TestMeasuredClockProfilesAndTampering(t *testing.T) {
	for _, rtt := range []int64{50, 100, 200, 320} {
		t.Run((time.Duration(rtt) * time.Millisecond).String(), func(t *testing.T) {
			var c clockEstimator
			for i, jitter := range []int64{0, 8, -4, 12} {
				sent := int64(10000 + i*2000)
				p := c.challenge(7, sent)
				client := float64(sent + (rtt+jitter)/2 - 9000)
				if err := c.observe(7, p.Nonce, client, sent+rtt+jitter); err != nil {
					t.Fatal(err)
				}
				if err := c.observe(7, p.Nonce, client, sent+rtt+jitter); err == nil {
					t.Fatal("nonce replay")
				}
			}
			rewind, err := c.rewind(7, 7110, 16110+rtt/2)
			if err != nil || rewind > 150 || rewind < 0 {
				t.Fatal(rewind, err)
			}
			if _, err = c.rewind(7, 7110, 16111+rtt/2); err == nil {
				t.Fatal("timestamp replay")
			}
			if _, err = c.rewind(7, 900000, 16200); err == nil {
				t.Fatal("future clock accepted")
			}
			if _, err = c.rewind(7, math.NaN(), 16200); err == nil {
				t.Fatal("NaN accepted")
			}
			if got, err := c.rewind(8, 1, 16200); err != nil || got != 0 {
				t.Fatal("new epoch used old offset", got, err)
			}
		})
	}
}
func TestRewindBoundAndHistoricalHit(t *testing.T) {
	e := newTest()
	e.targets = []target{{ID: 100, Kind: "fast", Born: 4500, X: 100, Y: 500, VX: 300}}
	e.Tick(time.UnixMilli(4800))
	e.Tick(time.UnixMilli(5000))
	r, err := e.Shoot("a", 1, Aim{.205, .5}, InputTiming{ReceivedAt: time.UnixMilli(5000), RewindMs: 150})
	if err != nil || r.TargetID != 100 {
		t.Fatal(r, err)
	}
	if _, err = e.Shoot("a", 2, Aim{}, InputTiming{ReceivedAt: time.UnixMilli(5300), RewindMs: 151}); err == nil {
		t.Fatal("oversize rewind")
	}
}

func TestClockFutureStaleAndUncalibratedNeverChooseHistory(t *testing.T) {
	var c clockEstimator
	if n, err := c.rewind(1, 1, 10000); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	challenge := c.challenge(1, 10000)
	if err := c.observe(1, challenge.Nonce, 1025, 10050); err != nil {
		t.Fatal(err)
	}
	if _, err := c.rewind(1, 1501, 10500); err == nil {
		t.Fatal("even slightly future mapped timestamp accepted")
	}
	if n, err := c.rewind(1, 25000, 35000); err != nil || n != 0 {
		t.Fatal("stale sample gained rewind", n, err)
	}
	stale := c.challenge(1, 40000)
	if err := c.observe(1, stale.Nonce, 32000, 41001); err == nil {
		t.Fatal("expired challenge accepted")
	}
}

func TestPresentationCalibrationUsesSameSelectedSampleAndExpiry(t *testing.T) {
	var c clockEstimator
	for _, sample := range []struct {
		at, rtt int64
		offset  float64
	}{{10000, 200, 9000}, {11000, 212, 9010}, {12000, 100, 9005}} {
		p := c.challenge(7, sample.at)
		client := float64(sample.at) + float64(sample.rtt)/2 - sample.offset
		if err := c.observe(7, p.Nonce, client, sample.at+sample.rtt); err != nil {
			t.Fatal(err)
		}
	}
	v, ok := c.calibration(12100)
	if !ok || v.RTTMs != 100 || v.OffsetMs != 9005 || v.RewindMs != 50 || v.MeasuredAt != 12100 || v.ExpiresAt != 27100 {
		t.Fatal(v, ok)
	}
	rewind, err := c.rewind(7, 3145, 12200)
	if err != nil || rewind != v.RewindMs {
		t.Fatal(rewind, err)
	}
	if _, ok = c.calibration(27101); ok {
		t.Fatal("expired presentation metadata")
	}
	c.bind(8)
	if _, ok = c.calibration(12200); ok {
		t.Fatal("epoch retained presentation metadata")
	}
}

func TestHistoricalPublicAimCannotResurrectDeadOrExpiredTarget(t *testing.T) {
	for _, expired := range []bool{false, true} {
		e := newTest()
		e.targets = []target{{ID: 100, Kind: "fast", Born: 4500, X: 500, Y: 500, VX: 0}}
		e.Tick(time.UnixMilli(4800))
		e.Tick(time.UnixMilli(5000))
		aim := Aim{.5, .5}
		if !expired {
			if r, err := shoot(e, 1, 5000, aim); err != nil || r.TargetID != 100 {
				t.Fatal(r, err)
			}
		}
		at := int64(5300)
		if expired {
			at = 7150
		}
		r, err := e.Shoot("a", 2, aim, InputTiming{ReceivedAt: time.UnixMilli(at), RewindMs: 150})
		if err != nil || r.TargetID == 100 {
			t.Fatal("historical aim resurrected expired/dead target", expired, r, err)
		}
	}
}
