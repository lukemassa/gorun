package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/zeebo/xxh3"

	log "github.com/lukemassa/clilog"
)

type ExecutableContext struct {
	MainPackage string
	Directory   string
}

type BuildCache struct {
	cacheDir              string
	compiler              compiler
	mu                    sync.RWMutex
	isCached              map[string]bool
	compilationInProgress map[string]*sync.Mutex
}

func NewBuildCache(cacheDir string, compiler compiler) *BuildCache {
	return &BuildCache{
		cacheDir:              cacheDir,
		isCached:              make(map[string]bool),
		mu:                    sync.RWMutex{},
		compiler:              compiler,
		compilationInProgress: map[string]*sync.Mutex{},
	}
}

type compiler interface {
	compile(e ExecutableContext, outputDir string) error
}

type defaultCompiler struct{}

func (d *defaultCompiler) compile(executableContext ExecutableContext, outputDir string) error {
	key := executableContext.Key()
	executable := filepath.Join(outputDir, key)
	cmd := exec.Command("go", "build", "-o", executable, executableContext.MainPackage)
	cmd.Dir = executableContext.Directory
	log.Infof("Running go build -o %s %s at %s", executable, executableContext.MainPackage, executableContext.Directory)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Failed to build: %s", string(output))
		// do better
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func (e ExecutableContext) Key() string {
	b := fmt.Appendf(nil, "%s\x00%s", e.MainPackage, e.Directory)
	return hashBytes(b)
}

func hashBytes(in []byte) string {
	sum := xxh3.Hash128(in)
	b := sum.Bytes()
	return hex.EncodeToString(b[:])
}

func (s *BuildCache) executablePath(e ExecutableContext) string {
	return filepath.Join(s.cacheDir, e.Key())
}

func (s *BuildCache) getExecutableFromContext(executableContext ExecutableContext) (string, error) {

	key := executableContext.Key()
	if s.isAlreadyCompiled(executableContext) {
		log.Infof("Skipping compilation for %+v (%s)", executableContext, key)
		return s.executablePath(executableContext), nil
	}
	log.Infof("Compiling for %+v (%s)", executableContext, key)
	err := s.compile(executableContext)

	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.isCached[key] = true
	s.mu.Unlock()
	return s.executablePath(executableContext), nil
}

func (s *BuildCache) isAlreadyCompiled(executableContext ExecutableContext) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isCached[executableContext.Key()]
}

func (s *BuildCache) compile(executableContext ExecutableContext) error {
	key := executableContext.Key()
	var compilationLock *sync.Mutex
	s.mu.Lock()
	compilationLock, ok := s.compilationInProgress[key]
	if !ok {
		compilationLock = &sync.Mutex{}
		s.compilationInProgress[key] = compilationLock
	}
	s.mu.Unlock()

	compilationLock.Lock()
	defer compilationLock.Unlock()

	return s.compiler.compile(executableContext, s.cacheDir)
}

func (s *BuildCache) recompile(executableContext ExecutableContext) error {
	key := executableContext.Key()
	log.Infof("Re-compiling compilation for %+v (%s)", executableContext, key)
	exectuable := s.executablePath(executableContext)

	err := os.Remove(exectuable)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file %s: %v", exectuable, err)
	}
	s.mu.Lock()
	delete(s.isCached, key)
	s.mu.Unlock()

	err = s.compile(executableContext)

	if err != nil {
		return fmt.Errorf("compiling for file %s: %v", exectuable, err)
	}

	s.mu.Lock()
	s.isCached[key] = true
	s.mu.Unlock()
	return nil
}
