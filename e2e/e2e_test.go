package e2e

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	result := runCLI(t, fstest.MapFS{
		"main.go": &fstest.MapFile{
			Data: []byte(`package main
import "fmt"

func main() {
	fmt.Println("Hello Gorun!")
}`),
		},
	}, "main.go")

	assert.Equal(t, "Hello Gorun!\n", result.Stdout)
	assert.Equal(t, 0, result.Code)
	assert.Contains(t, result.Stderr, "Compiled context for")
}
