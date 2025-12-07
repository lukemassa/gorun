package server

import (
	"log"
	"net"
	"net/http"
	"os"
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
