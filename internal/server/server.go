package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

const DefaultSock = "/tmp/api.sock"

type Server struct {
	sock string
}

func NewServer(sock string) *Server {
	return &Server{
		sock: sock,
	}
}

func (s *Server) Run() {

	_ = os.Remove(s.sock)

	l, err := net.Listen("unix", s.sock)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/bin/cat"))
	})

	srv := &http.Server{
		Handler: mux,
	}
	log.Printf("Starting server at %s", s.sock)

	// Serve blocks; if you want cancellation use srv.Serve(l) in a goroutine
	if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (s *Server) Start() (stop func(), err error) {

	_ = os.Remove(s.sock)

	l, err := net.Listen("unix", s.sock)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/bin/cat"))
	})

	srv := &http.Server{
		Handler: mux,
	}
	go srv.Serve(l)

	// wait until the server accepts connections
	serverCameUp := false
	for i := 0; i < 100; i++ {
		conn, err := net.Dial("unix", s.sock)
		if err == nil {
			conn.Close()
			serverCameUp = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !serverCameUp {
		srv.Shutdown(context.Background())
		return nil, errors.New("server did not start up")
	}

	return func() {
		_ = srv.Shutdown(context.Background())
	}, nil
}
