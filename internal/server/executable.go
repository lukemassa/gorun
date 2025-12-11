package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/lukemassa/clilog"
)

type ExecutableContext struct {
	MainPackage string
	Directory   string
}

func (e ExecutableContext) Key() string {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	// TODO: We call this a lot, should I use a simpler hashing algo?
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Server) executablePath(e ExecutableContext) string {
	return filepath.Join(s.cacheDir, e.Key())
}

func (s *Server) getExecutableFromContext(executableContext ExecutableContext) (string, error) {

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

	s.buildCacheMu.Lock()
	s.buildCache[key] = true
	s.buildCacheMu.Unlock()
	return s.executablePath(executableContext), nil
}

func (s *Server) isAlreadyCompiled(executableContext ExecutableContext) bool {
	s.buildCacheMu.RLock()
	defer s.buildCacheMu.RUnlock()
	return s.buildCache[executableContext.Key()]
}

func (s *Server) compile(executableContext ExecutableContext) error {
	exectuable := s.executablePath(executableContext)
	cmd := exec.Command("go", "build", "-o", exectuable, executableContext.MainPackage)
	cmd.Dir = executableContext.Directory
	log.Infof("Running go build -o %s %s at %s", exectuable, executableContext.MainPackage, executableContext.Directory)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Failed to build: %s", string(output))
		// do better
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func (s *Server) recompile(executableContext ExecutableContext) error {
	key := executableContext.Key()
	log.Infof("Re-compiling compilation for %+v (%s)", executableContext, key)
	exectuable := s.executablePath(executableContext)

	err := os.Remove(exectuable)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file %s: %v", exectuable, err)
	}
	s.buildCacheMu.Lock()
	delete(s.buildCache, key)
	s.buildCacheMu.Unlock()

	err = s.compile(executableContext)

	if err != nil {
		return fmt.Errorf("compiling for file %s: %v", exectuable, err)
	}

	s.buildCacheMu.Lock()
	s.buildCache[key] = true
	s.buildCacheMu.Unlock()
	return nil
}
