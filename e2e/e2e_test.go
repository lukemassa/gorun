package e2e

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	workingDir := t.TempDir()

	writeFS(t, fstest.MapFS{
		"main.go": &fstest.MapFile{
			Data: []byte(`package main
import "fmt"

func main() {
	fmt.Println("Hello Gorun!")
}`),
		},
	}, workingDir)

	result := runCLI(t, workingDir, "main.go")

	assert.Equal(t, "Hello Gorun!\n", result.Stdout)
	assert.Equal(t, 0, result.Code)
	assert.Contains(t, result.Stderr, "Compiled context for")
}

func TestCache(t *testing.T) {
	workingDir := t.TempDir()

	writeFS(t, fstest.MapFS{
		"main.go": &fstest.MapFile{
			Data: []byte(`package main
import "fmt"

func main() {
	fmt.Println("Hello Gorun!")
}`),
		},
	}, workingDir)

	runCLI(t, workingDir, "main.go")

	writeFS(t, fstest.MapFS{
		"main.go": &fstest.MapFile{
			Data: []byte(`package main
import "fmt"

func main() {
	fmt.Println("Something else!")
}`),
		},
	}, workingDir)

	result := runCLI(t, workingDir, "main.go")

	// This should still get the result of the first run, since it's still cached

	assert.Equal(t, "Hello Gorun!\n", result.Stdout)
	assert.Equal(t, 0, result.Code)
	assert.Contains(t, result.Stderr, "Compiled context for")
}
