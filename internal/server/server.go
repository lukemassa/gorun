package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	sock string
	srv  *http.Server
}

type ExecutableRequest struct {
	MainPackage string
	Env         []string
}

type ExecutableResponse struct {
	Executable string
}

type ExecutableContext struct {
	MainPackage string
	Directory   string
}

func (e ExecutableContext) Key() (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func valueFromEnv(key string, env []string) string {
	for i := 0; i < len(env); i++ {
		if strings.HasPrefix(env[i], key+"=") {
			return env[i][len(key)+1:]
		}
	}
	return ""
}

func (s *Server) handleExecutable(w http.ResponseWriter, r *http.Request) {
	var req ExecutableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Failed to parse json: %v", err)
		return
	}

	log.Printf("Requested translation of %s", req.MainPackage)
	executableContext := ExecutableContext{
		MainPackage: req.MainPackage,
		Directory:   valueFromEnv("PWD", req.Env),
	}
	newCommand, err := s.getExecutableFromContext(executableContext)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Failed to translate: %v", err)
		return
	}
	resp := ExecutableResponse{
		Executable: newCommand,
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

func (s *Server) getExecutableFromContext(executableContext ExecutableContext) (string, error) {

	key, err := executableContext.Key()
	if err != nil {
		return "", err
	}
	exectuable := filepath.Join("/tmp", key)

	cmd := exec.Command("go", "build", "-o", exectuable, executableContext.MainPackage)
	cmd.Dir = executableContext.Directory
	log.Printf("Running go build -o %s %s at %s", exectuable, executableContext.MainPackage, executableContext.Directory)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return exectuable, nil
}

func NewServer(sock string) *Server {

	s := &Server{
		sock: sock,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/command", s.handleExecutable)

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

	log.Printf("Starting server at %s", s.sock)

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
