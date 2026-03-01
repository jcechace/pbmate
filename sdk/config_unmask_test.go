package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"

	"github.com/percona/percona-backup-mongodb/pbm/storage"
)

// testCredentials is a minimal struct mirroring PBM's credential structs
// for testing unmaskYAML without constructing a full config.Config.
type testCredentials struct {
	AccessKey storage.MaskedString `bson:"access-key" yaml:"access-key,omitempty"`
	SecretKey storage.MaskedString `bson:"secret-key" yaml:"secret-key,omitempty"`
}

// testStorage mirrors a simplified StorageConf for testing.
type testStorage struct {
	Type        string          `bson:"type" yaml:"type"`
	Bucket      string          `bson:"bucket" yaml:"bucket"`
	Credentials testCredentials `bson:"credentials" yaml:"credentials"`
}

// testConfig mirrors Config with metadata fields for testing filtering.
type testConfig struct {
	Name    string      `bson:"name,omitempty" yaml:"name,omitempty"`
	Profile bool        `bson:"profile,omitempty" yaml:"profile,omitempty"`
	Epoch   int64       `bson:"epoch" yaml:"-"`
	Storage testStorage `bson:"storage" yaml:"storage"`
}

func TestUnmaskYAML(t *testing.T) {
	tests := []struct {
		name       string
		input      any
		wantKeys   []string // top-level keys expected in order
		wantAbsent []string // top-level keys that must NOT appear
		checkValue func(t *testing.T, slice yaml.MapSlice)
	}{
		{
			name: "credentials unmasked",
			input: testConfig{
				Storage: testStorage{
					Type:   "s3",
					Bucket: "my-bucket",
					Credentials: testCredentials{
						AccessKey: "AKIAIOSFODNN7EXAMPLE",
						SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					},
				},
			},
			wantKeys: []string{"storage"},
			checkValue: func(t *testing.T, slice yaml.MapSlice) {
				storageSlice := findMapItem(t, slice, "storage")
				credsSlice := findMapItem(t, storageSlice, "credentials")
				assertMapValue(t, credsSlice, "access-key", "AKIAIOSFODNN7EXAMPLE")
				assertMapValue(t, credsSlice, "secret-key", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			},
		},
		{
			name: "metadata keys filtered",
			input: testConfig{
				Name:    "my-profile",
				Profile: true,
				Epoch:   12345,
				Storage: testStorage{Type: "s3", Bucket: "b"},
			},
			wantKeys:   []string{"storage"},
			wantAbsent: []string{"name", "profile", "epoch"},
		},
		{
			name: "field order preserved",
			input: testStorage{
				Type:   "s3",
				Bucket: "my-bucket",
				Credentials: testCredentials{
					AccessKey: "key",
					SecretKey: "secret",
				},
			},
			wantKeys: []string{"type", "bucket", "credentials"},
		},
		{
			name: "empty credentials omitted by bson omitempty",
			input: testConfig{
				Storage: testStorage{
					Type:   "filesystem",
					Bucket: "b",
					Credentials: testCredentials{
						AccessKey: "",
						SecretKey: "",
					},
				},
			},
			wantKeys: []string{"storage"},
			checkValue: func(t *testing.T, slice yaml.MapSlice) {
				storageSlice := findMapItem(t, slice, "storage")
				credsSlice := findMapItem(t, storageSlice, "credentials")
				// Empty MaskedString marshals as empty string via BSON.
				assertMapValue(t, credsSlice, "access-key", "")
				assertMapValue(t, credsSlice, "secret-key", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmaskYAML(tt.input)
			require.NoError(t, err)

			var slice yaml.MapSlice
			require.NoError(t, yaml.Unmarshal(got, &slice))

			if tt.wantKeys != nil {
				keys := mapSliceKeys(slice)
				assert.Equal(t, tt.wantKeys, keys, "top-level key order")
			}

			for _, absent := range tt.wantAbsent {
				for _, item := range slice {
					assert.NotEqual(t, absent, item.Key, "key %q should be filtered", absent)
				}
			}

			if tt.checkValue != nil {
				tt.checkValue(t, slice)
			}
		})
	}
}

func TestBsonDToMapSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    bson.D
		wantKeys []string
	}{
		{
			name: "nested documents",
			input: bson.D{
				{Key: "outer", Value: bson.D{
					{Key: "inner", Value: "value"},
				}},
			},
			wantKeys: []string{"outer"},
		},
		{
			name: "arrays",
			input: bson.D{
				{Key: "items", Value: bson.A{"a", "b", "c"}},
			},
			wantKeys: []string{"items"},
		},
		{
			name:     "empty document",
			input:    bson.D{},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := bsonDToMapSlice(tt.input)
			assert.Equal(t, tt.wantKeys, mapSliceKeys(slice))
		})
	}
}

func TestBsonToYAML(t *testing.T) {
	tests := []struct {
		name  string
		input any
		check func(t *testing.T, result any)
	}{
		{
			name:  "bson.D becomes MapSlice",
			input: bson.D{{Key: "k", Value: "v"}},
			check: func(t *testing.T, result any) {
				slice, ok := result.(yaml.MapSlice)
				require.True(t, ok, "expected yaml.MapSlice")
				assert.Len(t, slice, 1)
				assert.Equal(t, "k", slice[0].Key)
				assert.Equal(t, "v", slice[0].Value)
			},
		},
		{
			name:  "bson.A becomes slice",
			input: bson.A{"x", int32(1), true},
			check: func(t *testing.T, result any) {
				arr, ok := result.([]any)
				require.True(t, ok, "expected []any")
				assert.Equal(t, []any{"x", int32(1), true}, arr)
			},
		},
		{
			name:  "primitives pass through",
			input: "hello",
			check: func(t *testing.T, result any) {
				assert.Equal(t, "hello", result)
			},
		},
		{
			name:  "nil passes through",
			input: nil,
			check: func(t *testing.T, result any) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bsonToYAML(tt.input)
			tt.check(t, result)
		})
	}
}

// --- test helpers ---

// mapSliceKeys extracts the key names from a yaml.MapSlice in order.
func mapSliceKeys(slice yaml.MapSlice) []string {
	keys := make([]string, len(slice))
	for i, item := range slice {
		keys[i] = item.Key.(string)
	}
	return keys
}

// findMapItem finds a nested yaml.MapSlice value by key within a parent MapSlice.
func findMapItem(t *testing.T, slice yaml.MapSlice, key string) yaml.MapSlice {
	t.Helper()
	for _, item := range slice {
		if item.Key == key {
			nested, ok := item.Value.(yaml.MapSlice)
			require.True(t, ok, "key %q value is not a MapSlice", key)
			return nested
		}
	}
	t.Fatalf("key %q not found in MapSlice", key)
	return nil
}

// assertMapValue asserts that a key in a MapSlice has the expected string value.
func assertMapValue(t *testing.T, slice yaml.MapSlice, key string, want string) {
	t.Helper()
	for _, item := range slice {
		if item.Key == key {
			assert.Equal(t, want, item.Value, "value for key %q", key)
			return
		}
	}
	t.Errorf("key %q not found in MapSlice", key)
}
