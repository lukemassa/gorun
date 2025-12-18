package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/lukemassa/clilog"
	"github.com/lukemassa/gorun/internal/build"
)

type Server struct {
	sock  string
	srv   *http.Server
	cache *build.Cache
}

type ExecutableRequest struct {
	MainPackage string
	Env         []string
}

type ExecutableResponse struct {
	Executable        string
	CompilationOutput string
}

func (s *Server) handleExecutable(w http.ResponseWriter, r *http.Request) {
	var req ExecutableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Failed to parse json: %v", err)
		return
	}

	log.Infof("Requested translation of %s", req.MainPackage)
	executableContext := build.Context{
		MainPackage: req.MainPackage,
		Directory:   valueFromEnv("PWD", req.Env),
	}
	newCommand, err := s.cache.GetExecutableFromContext(executableContext)
	resp := ExecutableResponse{
		Executable: newCommand,
	}
	if err != nil {
		resp.Executable = ""
		resp.CompilationOutput = err.Error()
	}
	respContent, err := json.Marshal(&resp)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Failed to json marshal result: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(respContent)

}

func (s *Server) handleDeleteExecutable(w http.ResponseWriter, r *http.Request) {
	var req ExecutableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Failed to parse json: %v", err)
		return
	}

	log.Infof("Requested deletion of %s", req.MainPackage)
	executableContext := build.Context{
		MainPackage: req.MainPackage,
		Directory:   valueFromEnv("PWD", req.Env),
	}
	err := s.cache.Recompile(executableContext)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Failed to recompile: %v", err)
		return
	}
	w.WriteHeader(200)
	fmt.Fprintf(w, "Recompiled %+v", executableContext)
}

func NewServer(sock, cacheDir string) *Server {

	s := &Server{
		sock:  sock,
		cache: build.NewCache(cacheDir, &build.DefaultCompiler{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /command", s.handleExecutable)
	mux.HandleFunc("DELETE /command", s.handleDeleteExecutable)

	s.srv = &http.Server{
		Handler: mux,
	}
	return s
}

func (s *Server) Run() {

	if err := s.serve(); err != nil {
		panic(err)
	}
}

func (s *Server) serve() (err error) {

	_ = os.Remove(s.sock)

	l, err := net.Listen("unix", s.sock)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Infof("Starting server at %s", s.sock)

	err = s.srv.Serve(l)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Start() (stop func(), err error) {

	go s.serve()

	stopFn := func() {
		_ = s.srv.Shutdown(context.Background())
	}

	for range 100 {
		if conn, err := net.Dial("unix", s.sock); err == nil {
			conn.Close()
			return stopFn, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, errors.New("server did not start up")
}

func valueFromEnv(key string, env []string) string {
	for i := range env {
		if strings.HasPrefix(env[i], key+"=") {
			return env[i][len(key)+1:]
		}
	}
	return ""
}
