package world

import _ "embed"

// Generated asset copies are checked against contracts by TestEmbeddedAssets.
//
//go:embed assets/map.v1.json
var MapAsset []byte

//go:embed assets/rules.v1.json
var RulesAsset []byte
