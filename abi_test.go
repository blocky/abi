package abi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/abi"
)

func TestHelloWorld(t *testing.T) {
	x, err := abi.HelloWorld()
	require.NoError(t, err)
	assert.Equal(t, x, "Hello, world!")
}
