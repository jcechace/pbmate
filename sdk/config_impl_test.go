package sdk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestCleanParseError(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		err     error
		wantMsg string
	}{
		{
			name:   "single yaml type error",
			prefix: "invalid config",
			err: &yaml.TypeError{
				Errors: []string{`line 3: field sdsstorage not found in type config.Config`},
			},
			wantMsg: "invalid config: line 3: field sdsstorage not found",
		},
		{
			name:   "multiple yaml type errors",
			prefix: "invalid profile config",
			err: &yaml.TypeError{
				Errors: []string{
					`line 3: field foo not found in type config.Config`,
					`line 7: field bar not found in type config.StorageConf`,
				},
			},
			wantMsg: "invalid profile config: line 3: field foo not found; line 7: field bar not found",
		},
		{
			name:    "non yaml type error passes through",
			prefix:  "invalid config",
			err:     fmt.Errorf("some other error"),
			wantMsg: "invalid config: some other error",
		},
		{
			name:   "yaml error without type suffix preserved",
			prefix: "invalid config",
			err: &yaml.TypeError{
				Errors: []string{`line 1: cannot unmarshal !!str into bool`},
			},
			wantMsg: "invalid config: line 1: cannot unmarshal !!str into bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanParseError(tt.prefix, tt.err)
			assert.EqualError(t, got, tt.wantMsg)
		})
	}
}
