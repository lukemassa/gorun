package server

import (
	"crypto/rand"
	"encoding/hex"
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
	cacheDir    string
	compiler    compiler
	mu          sync.RWMutex
	executables map[string]*Executable
}

type Executable struct {
	currentPath  string
	buildBarrier sync.Mutex
}

func NewBuildCache(cacheDir string, compiler compiler) *BuildCache {
	return &BuildCache{
		cacheDir:    cacheDir,
		mu:          sync.RWMutex{},
		compiler:    compiler,
		executables: make(map[string]*Executable),
	}
}

type compiler interface {
	compile(e ExecutableContext, outputFile string) error
}

type defaultCompiler struct{}

func (d *defaultCompiler) compile(executableContext ExecutableContext, outputFile string) error {
	cmd := exec.Command("go", "build", "-o", outputFile, executableContext.MainPackage)
	cmd.Dir = executableContext.Directory
	log.Infof("Running go build -o %s %s at %s", outputFile, executableContext.MainPackage, executableContext.Directory)
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

func (s *BuildCache) getExecutableFromContext(executableContext ExecutableContext) (string, error) {

	key := executableContext.Key()
	var e *Executable
	s.mu.Lock()
	e, ok := s.executables[key]
	if !ok {
		e = &Executable{
			buildBarrier: sync.Mutex{},
		}
		s.executables[key] = e
	}
	s.mu.Unlock()

	e.buildBarrier.Lock()
	log.Info("inside build lock")
	defer e.buildBarrier.Unlock()
	if e.currentPath != "" {
		log.Infof("Path found %s in cache", e.currentPath)
		return e.currentPath, nil
	}
	log.Infof("Must compile for %v", executableContext)
	newPath, err := s.compile(executableContext)
	if err != nil {
		return "", err
	}
	e.currentPath = newPath
	return newPath, nil
}

func (s *BuildCache) compile(executableContext ExecutableContext) (string, error) {
	key := executableContext.Key()

	outputDir := filepath.Join(s.cacheDir, key)

	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return "", err
	}

	var randIdentifier []byte
	_, err = rand.Read(randIdentifier)
	if err != nil {
		return "", err
	}

	newPath := filepath.Join(outputDir, hashBytes(randIdentifier))
	err = s.compiler.compile(executableContext, newPath)
	if err != nil {
		return "", err
	}
	return newPath, nil
}

func (s *BuildCache) recompile(executableContext ExecutableContext) error {
	key := executableContext.Key()
	log.Infof("Re-compiling compilation for %+v (%s)", executableContext, key)
	s.mu.Lock()
	e := s.executables[key]
	s.mu.Lock()

	e.buildBarrier.Lock()
	defer e.buildBarrier.Unlock()
	_, err := s.compile(executableContext)
	return err
}
