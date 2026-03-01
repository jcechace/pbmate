package sdk

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"
)

// TODO(pbm-fix): This file is a workaround for PBM's storage.MaskedString type,
// which has a MarshalYAML method that unconditionally replaces credentials with
// "***". This makes YAML roundtripping impossible — editing and re-applying
// config destroys credentials. The Percona Operator hit the same issue.
//
// The proper fix is for PBM to treat masking as a presentation concern (in the
// CLI's output path) rather than baking it into the type's serialization.
//
// Workaround: MaskedString has no MarshalBSON method, so bson.Marshal preserves
// real credential values. We marshal to BSON, unmarshal into an ordered bson.D,
// convert to yaml.MapSlice (preserving field order), and marshal to YAML.
//
// Known limitation — omitempty discrepancy: Several PBM struct fields have
// omitempty on their yaml tag but not on their bson tag (BackupConf, RestoreConf,
// Azure config). This means the BSON roundtrip may emit zero-value fields
// (e.g. "batchSize: 0") that yaml.Marshal on the struct would normally omit.
// These are valid config values and config.Parse accepts them. An upstream PBM
// fix to align bson tags with yaml tags is pending.

// metadataKeys are top-level PBM Config document fields that should not appear
// in user-facing YAML output. These are filtered during unmaskYAML conversion:
//   - "epoch": internal PBM coordination timestamp (yaml:"-" on the struct)
//   - "name": profile name, empty for main config (yaml:",omitempty")
//   - "profile": IsProfile bool, false for main config (yaml:",omitempty")
var metadataKeys = map[string]struct{}{
	"epoch":   {},
	"name":    {},
	"profile": {},
}

// unmaskYAML marshals v to YAML with real credential values, bypassing
// MaskedString.MarshalYAML. See the TODO(pbm-fix) block at the top of this
// file for full context.
func unmaskYAML(v any) ([]byte, error) {
	raw, err := bson.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("bson marshal: %w", err)
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("bson unmarshal: %w", err)
	}

	// Filter top-level metadata keys that yaml tags would normally exclude.
	filtered := make(bson.D, 0, len(doc))
	for _, e := range doc {
		if _, skip := metadataKeys[e.Key]; !skip {
			filtered = append(filtered, e)
		}
	}

	mapSlice := bsonDToMapSlice(filtered)

	out, err := yaml.Marshal(mapSlice)
	if err != nil {
		return nil, fmt.Errorf("yaml marshal: %w", err)
	}
	return out, nil
}

// bsonDToMapSlice converts an ordered bson.D to a yaml.MapSlice,
// preserving field order. Nested documents and arrays are converted
// recursively via bsonToYAML.
func bsonDToMapSlice(d bson.D) yaml.MapSlice {
	slice := make(yaml.MapSlice, len(d))
	for i, e := range d {
		slice[i] = yaml.MapItem{
			Key:   e.Key,
			Value: bsonToYAML(e.Value),
		}
	}
	return slice
}

// bsonToYAML converts a BSON value to its YAML-compatible equivalent.
// bson.D becomes yaml.MapSlice, bson.A becomes []any, and primitive
// types (string, int32, int64, float64, bool, nil) pass through.
func bsonToYAML(v any) any {
	switch val := v.(type) {
	case bson.D:
		return bsonDToMapSlice(val)
	case bson.A:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = bsonToYAML(item)
		}
		return out
	default:
		return val
	}
}
