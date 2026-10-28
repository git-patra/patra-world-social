package race

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestTrackAssetAndChecksumBindSimulatorGeometry(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/race-track.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, trackAsset) {
		t.Fatal("embedded race track drift")
	}
	var object map[string]any
	if err = json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	checksum := object["checksum"]
	delete(object, "checksum")
	canonical, _ := json.Marshal(object)
	if fmt.Sprintf("%x", sha256.Sum256(canonical)) != checksum || DefaultRules().TrackChecksum != checksum {
		t.Fatal("geometry not identified by recorded rules")
	}
}
