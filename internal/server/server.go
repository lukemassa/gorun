package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	DefaultSock string
	CacheDir    string
)

type Server struct {
	sock string
	srv  *http.Server
}

type CommandRequest struct {
	Cmd string
	Env []string
}

type CommandResponse struct {
	Cmd string
}

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	CacheDir = filepath.Join(userCacheDir, "gorun-cache")
	err = os.MkdirAll(CacheDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	DefaultSock = filepath.Join(CacheDir, "gorun.sock")
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Failed to parse json: %v", err)
		return
	}

	log.Printf("Requested translation of %s", req.Cmd)
	newCommand, err := s.getCommand(req.Cmd, req.Env)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Failed to translate: %v", err)
		return
	}
	resp := CommandResponse{
		Cmd: newCommand,
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

func (s *Server) getCommand(initialCommand string, env []string) (string, error) {
	// TODO: Implement
	return "/bin/cat", nil
}

func NewServer(sock string) *Server {

	s := &Server{
		sock: sock,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/command", s.handleCommand)

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
