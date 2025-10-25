package abi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNonZero(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input []byte
		want  bool
	}{
		{name: "empty", input: []byte{}, want: false},
		{name: "one zero", input: []byte{0}, want: false},
		{name: "multiple zeros", input: []byte{0, 0, 0}, want: false},
		{name: "non-zero", input: []byte{1}, want: true},
		{name: "non-zero with zeros", input: []byte{0, 1, 0}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := isNonZero(tc.input)
			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPadRight(t *testing.T) {
	fourBytes := []byte{15, 16, 23, 42}

	for _, tc := range []struct {
		name      string
		input     []byte
		targetLen int
		want      []byte
	}{
		{
			name:      "more than 1 smaller than target length",
			input:     fourBytes,
			targetLen: 10,
			want:      append(fourBytes, 0, 0, 0, 0, 0, 0),
		}, {
			name:      "1 smaller than target length",
			input:     fourBytes,
			targetLen: 5,
			want:      append(fourBytes, 0),
		}, {
			name:      "same as target length",
			input:     fourBytes,
			targetLen: 4,
			want:      fourBytes,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := padRight(tc.input, tc.targetLen)

			// then
			require.NoError(t, err)
			assert.Len(t, got, tc.targetLen)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("input is larger than target", func(t *testing.T) {
		// given
		input := fourBytes
		// when
		_, err := padRight(input, 3)
		// then
		assert.ErrorContains(t, err, "smaller than input")
	})
}

func TestNextMultipleOf32(t *testing.T) {
	for _, tc := range []struct {
		start int
		end   int
		want  int
	}{
		{start: -33, end: -32, want: -32},
		{start: -31, end: 0, want: 0},
		{start: 1, end: 32, want: 32},
		{start: 33, end: 64, want: 64},
		{start: 65, end: 70, want: 96},
	} {
		tcName := fmt.Sprintf("start: %d, end: %d", tc.start, tc.end)
		t.Run(tcName, func(t *testing.T) {
			for i := tc.start; i <= tc.end; i++ {
				// when
				got := nextMultipleOf32(i)
				// then
				assert.Equal(t, got, tc.want, "using value %d", i)
			}
		})
	}
}
