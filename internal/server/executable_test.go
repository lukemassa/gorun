package server

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCompiler struct {
}

func (m *mockCompiler) compile(e ExecutableContext, outputDir string) error {
	key := e.Key()
	executable := filepath.Join(outputDir, key)
	_, err := os.Create(executable)
	return err
}

func TestExecutableFromContext(t *testing.T) {
	dir := t.TempDir()
	cache := BuildCache{
		cacheDir:   dir,
		isCached:   make(map[string]bool),
		isCachedMU: &sync.RWMutex{},
		compiler:   &mockCompiler{},
	}

	e := ExecutableContext{}
	key := e.Key()
	executable, err := cache.getExecutableFromContext(e)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, key), executable)
	assert.FileExists(t, executable)
}
