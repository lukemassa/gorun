package server

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockCompiler struct {
	mu         sync.Mutex
	inProgress map[string]int
}

func newMockCompiler() *mockCompiler {
	return &mockCompiler{
		inProgress: make(map[string]int),
		mu:         sync.Mutex{},
	}
}

func (m *mockCompiler) compile(e ExecutableContext, outputDir string) error {
	log.Print("Doing a mock compile!")
	key := e.Key()

	// Simulates compilation, and panics if two compiles are called simultaneously
	m.mu.Lock()
	m.inProgress[key]++
	if m.inProgress[key] > 1 {
		m.mu.Unlock()
		panic("concurrent compile")
	}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.inProgress[key]--
		m.mu.Unlock()
	}()

	executable := filepath.Join(outputDir, key)

	time.Sleep(10 * time.Millisecond) // Simulate real compilation
	_, err := os.Create(executable)

	return err
}

func TestExecutableFromContext(t *testing.T) {
	dir := t.TempDir()
	compiler := newMockCompiler()
	cache := NewBuildCache(dir, compiler)

	e := ExecutableContext{}
	key := e.Key()
	executable, err := cache.getExecutableFromContext(e)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, key), executable)
	assert.FileExists(t, executable)
}

func TestPreventSimultaneousCompilation(t *testing.T) {
	dir := t.TempDir()
	compiler := newMockCompiler()
	cache := NewBuildCache(dir, compiler)

	e := ExecutableContext{}

	wg := sync.WaitGroup{}
	// mock compiler should blow up if they are called simultaneously
	for range 10 {
		wg.Go(func() {
			cache.compile(e)
		})
	}
	wg.Wait()
}
