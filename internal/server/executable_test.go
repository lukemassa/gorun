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

func (m *mockCompiler) compile(e ExecutableContext, outputPath string) error {
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

	time.Sleep(10 * time.Millisecond) // Simulate real compilation
	_, err := os.Create(outputPath)

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
	assert.Equal(t, filepath.Join(dir, key), filepath.Dir(executable))
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
			cache.getExecutableFromContext(e)
		})
	}
	wg.Wait()
}

type blockingCompiler struct {
	started chan struct{}
	proceed chan struct{}
}

func (b *blockingCompiler) compile(e ExecutableContext, outputFile string) error {
	// Signal that compile has started (and recompile already removed the file)
	b.started <- struct{}{}

	// Block until test allows us to continue
	<-b.proceed

	_, err := os.Create(outputFile)
	return err
}
func TestReturnedPathRemainsUsableDuringRecompile(t *testing.T) {
	dir := t.TempDir()

	c := &blockingCompiler{
		started: make(chan struct{}, 1),
		proceed: make(chan struct{}, 1),
	}

	cache := NewBuildCache(dir, c)
	e := ExecutableContext{}

	// Allow the initial compile to finish
	c.proceed <- struct{}{}

	path, err := cache.getExecutableFromContext(e)
	assert.NoError(t, err)

	// Drain the "started" signal from the initial compile
	<-c.started

	// Start a recompile AFTER the path is handed out
	go func() {
		_ = cache.recompile(e)
	}()

	// Wait until recompile has removed the file and is blocked in compile
	<-c.started

	// Invariant: the returned path should still be usable
	assert.FileExists(t, path)
}
