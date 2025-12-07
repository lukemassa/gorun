package e2e

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	result := runCLI(t, fstest.MapFS{
		"hello.txt": &fstest.MapFile{
			Data: []byte("Contents!"),
		},
	}, "hello.txt")

	assert.Equal(t, "Contents!", result.Stdout)
	assert.Equal(t, 0, result.Code)
	assert.Contains(t, result.Stderr, "Translated initial command")
}
