package rpc

import (
	"log"
	"net"
	"net/http"
	"os"
)

const sock = "/tmp/api.sock"

type Server struct{}

func (s Server) Run() {

	_ = os.Remove(sock)

	l, err := net.Listen("unix", sock)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/bin/echo"))
	})

	srv := &http.Server{
		Handler: mux,
	}
	log.Printf("Starting server at %s", sock)

	// Serve blocks; if you want cancellation use srv.Serve(l) in a goroutine
	if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
