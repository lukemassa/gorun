package build

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

type Context struct {
	MainPackage string
	Directory   string
}

type Cache struct {
	cacheDir    string
	compiler    compiler
	mu          sync.Mutex
	executables map[string]*executable
}

type executable struct {
	currentPath  string
	buildBarrier sync.Mutex
}

func NewCache(cacheDir string, compiler compiler) *Cache {
	return &Cache{
		cacheDir:    cacheDir,
		compiler:    compiler,
		executables: make(map[string]*executable),
	}
}

type compiler interface {
	compile(e Context, outputFile string) error
}

type DefaultCompiler struct{}

func (d *DefaultCompiler) compile(executableContext Context, outputFile string) error {
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

func (e Context) Key() string {
	b := fmt.Appendf(nil, "%s\x00%s", e.MainPackage, e.Directory)
	return hashBytes(b)
}

func hashBytes(in []byte) string {
	sum := xxh3.Hash128(in)
	b := sum.Bytes()
	return hex.EncodeToString(b[:])
}

func (s *Cache) GetExecutableFromContext(executableContext Context) (string, error) {

	key := executableContext.Key()
	var e *executable
	s.mu.Lock()
	e, ok := s.executables[key]
	if !ok {
		e = &executable{}
		s.executables[key] = e
	}
	s.mu.Unlock()

	e.buildBarrier.Lock()
	if e.currentPath != "" {
		e.buildBarrier.Unlock()
		log.Infof("Path found %s in cache", e.currentPath)
		return e.currentPath, nil
	}
	log.Infof("Must compile for %v", executableContext)
	defer e.buildBarrier.Unlock()
	newPath, err := s.compile(executableContext)
	if err != nil {
		return "", err
	}
	e.currentPath = newPath
	return newPath, nil
}

func randomHex32() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *Cache) compile(executableContext Context) (string, error) {
	key := executableContext.Key()

	outputDir := filepath.Join(s.cacheDir, key)

	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return "", err
	}
	filename, err := randomHex32()
	if err != nil {
		return "", err
	}

	newPath := filepath.Join(outputDir, filename)
	err = s.compiler.compile(executableContext, newPath)
	if err != nil {
		return "", err
	}
	return newPath, nil
}

func (s *Cache) Recompile(executableContext Context) error {
	key := executableContext.Key()
	log.Infof("Re-compiling compilation for %+v (%s)", executableContext, key)
	s.mu.Lock()
	e, ok := s.executables[key]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("attempted recompilation when there was no initial compile")
	}

	e.buildBarrier.Lock()
	defer e.buildBarrier.Unlock()
	newPath, err := s.compile(executableContext)
	if err != nil {
		return err
	}
	e.currentPath = newPath
	return err
}
