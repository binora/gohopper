package osm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"00:10", 10 * 60},
		{"35", 35 * 60},
		{"01:10", 70 * 60},
		{"01:10:02", 70*60 + 2},
		{"", 0},
		{"20:00", 60 * 20 * 60},
		{"0:20:00", 20 * 60},
		{"02:20:02", (60*2+20)*60 + 2},
		// ISO 8601: two months (31 days each with Java-compatible reference date)
		{"P2M", 62 * 24 * 60 * 60},
		// ISO 8601: two minutes
		{"PT2M", 2 * 60},
		// ISO 8601: complex
		{"PT5H12M36S", (5*60+12)*60 + 36},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseDuration(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDurationErrors(t *testing.T) {
	invalid := []string{
		"PT5h12m36s", // lowercase
		"oh",         // invalid
		"01:10:2",    // seconds not 2 digits
	}

	for _, s := range invalid {
		t.Run(s, func(t *testing.T) {
			_, err := parseDuration(s)
			assert.Error(t, err)
		})
	}
}
