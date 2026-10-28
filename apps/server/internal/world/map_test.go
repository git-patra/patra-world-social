package world

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestSharedMap(t *testing.T) {
	data, e := os.ReadFile("../../../../contracts/map.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	m, e := LoadMap(data)
	if e != nil || m.Width != 64 {
		t.Fatalf("%+v %v", m, e)
	}
	for _, b := range m.Buildings {
		if !m.Reachable(m.Spawn, b.Entrance) {
			t.Fatal(b.ID)
		}
	}
	for _, bad := range [][]byte{bytes.Replace(data, []byte(`"v": 1`), []byte(`"v": 2`), 1), []byte(`{"v":1}`), bytes.Replace(data, []byte(`"x": 32`), []byte(`"x": null`), 1)} {
		if _, e := LoadMap(bad); e == nil {
			t.Fatal("accepted invalid map")
		}
	}
}
func TestSharedRules(t *testing.T) {
	data, e := os.ReadFile("../../../../contracts/rules.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := LoadRules(data); e != nil {
		t.Fatal(e)
	}
}

func TestInvalidSemanticMaps(t *testing.T) {
	raw, e := os.ReadFile("../../../../contracts/maps.invalid.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fixtures); e != nil {
		t.Fatal(e)
	}
	for name, data := range fixtures {
		if _, e = LoadMap(data); e == nil {
			t.Error("accepted", name)
		}
	}
}

func TestMapIntegrityRejectsTamperedPayload(t *testing.T) {
	raw, e := os.ReadFile("../../../../contracts/map.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	raw = bytes.Replace(raw, []byte(`"id": "kampung-senja-v2"`), []byte(`"id": "untrusted-village"`), 1)
	if _, e = LoadMap(raw); e == nil {
		t.Fatal("tampered map accepted")
	}
}
func TestEmbeddedAssets(t *testing.T) {
	for name, embedded := range map[string][]byte{"map.v1.json": MapAsset, "rules.v1.json": RulesAsset} {
		raw, e := os.ReadFile("../../../../contracts/" + name)
		if e != nil || !bytes.Equal(raw, embedded) {
			t.Fatalf("embedded asset drift: %s", name)
		}
	}
	if _, e := LoadMap(MapAsset); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadRules(RulesAsset); e != nil {
		t.Fatal(e)
	}
}
