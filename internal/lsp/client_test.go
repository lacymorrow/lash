package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlesFileAlwaysTrue(t *testing.T) {
	client := &Client{}
	require.True(t, client.HandlesFile("test.go"))
	require.True(t, client.HandlesFile("/path/to/file.tsx"))
	require.True(t, client.HandlesFile("script.sh"))
}
