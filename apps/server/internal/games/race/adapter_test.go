package race

import (
	"patra-world-social/server/internal/games/model"
	"reflect"
	"testing"
	"time"
)

func TestTimedPresencePublishesNaturalFinish(t *testing.T) {
	f := Factory(model.SystemClock{})
	v, err := f(startFixture(1))
	if err != nil {
		t.Fatal(err)
	}
	a := v.(*Adapter)
	a.Start(4000)
	a.PresenceAt("a", "disconnect", time.UnixMilli(304000))
	if _, err = a.Result(); err == nil || a.HasEvents() {
		t.Fatal("metadata installation finalized or published before owner advance")
	}
	a.Tick(time.UnixMilli(304000))
	if _, err = a.Result(); err != nil {
		t.Fatal("natural limit lost to disconnect")
	}
	events := a.DrainEvents()
	if len(events) != 1 || events[0].Type != "race.finished" {
		t.Fatal("terminal snapshot lost inside presence advance", events)
	}
}

func TestOrderedDeparturePublishesNaturalFinish(t *testing.T) {
	for _, kind := range []string{"resume", "expired", "leave"} {
		t.Run(kind, func(t *testing.T) {
			v, err := Factory(model.SystemClock{})(startFixture(1))
			if err != nil {
				t.Fatal(err)
			}
			a := v.(*Adapter)
			a.Start(4000)
			a.PresenceAt("a", "disconnect", time.UnixMilli(289000))
			a.PresenceAt("a", kind, time.UnixMilli(304000))
			result, err := a.Result()
			if err != nil || result.FinishedAt != 304000 {
				t.Fatal("natural finish lost at grace boundary", result, err)
			}
			events := a.DrainEvents()
			if len(events) != 1 || events[0].Type != "race.finished" {
				t.Fatal("terminal presence event lost", events)
			}
			a.Tick(time.UnixMilli(304500))
			if a.HasEvents() {
				t.Fatal("terminal event repeated")
			}
		})
	}
}

func TestAdapterShortenedCutoffIgnoresLaterDisconnect(t *testing.T) {
	e, baseline := cutoffFixture(true, false), cutoffFixture(true, false)
	a := &Adapter{start: e.start, clock: model.SystemClock{}, engine: e}
	a.Tick(time.UnixMilli(54950))
	a.DrainEvents()
	baseline.Tick(time.UnixMilli(54950))
	a.PresenceAt("a", "disconnect", time.UnixMilli(55001))
	if a.HasEvents() {
		t.Fatal("metadata published before advance")
	}
	a.Tick(time.UnixMilli(55001))
	baseline.Tick(time.UnixMilli(55001))
	got, err := a.Result()
	want, _ := baseline.Result()
	if err != nil || got.FinishedAt != 55000 || got.Participants[0].BotAssisted || !reflect.DeepEqual(got, want) {
		t.Fatal("shortened cutoff result changed", got, want, err)
	}
	events := a.DrainEvents()
	if len(events) != 1 || events[0].Type != "race.finished" || !reflect.DeepEqual(events[0].Payload.(PublicSnapshot).Result, &got) {
		t.Fatal("shortened terminal event lost or changed", events)
	}
	a.Tick(time.UnixMilli(55100))
	if a.HasEvents() {
		t.Fatal("shortened terminal event repeated")
	}
}
