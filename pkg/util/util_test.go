package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestEscapeQuotes(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input:    "\"",
			expected: "\\\"",
		},
		{
			input:    `"quoted"`,
			expected: `\"quoted\"`,
		},
	}
	for _, tc := range testcases {
		t.Run("", func(t *testing.T) {
			actual := EscapeQuotes(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestEscapeQuotesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`.*".*`).Draw(t, "String with quote").(string)

		out := EscapeQuotes(s)

		require.NotRegexp(t, `($")|([^\\]")`, out)
	})
}
