package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"http-protocol/internal/request"
)

const (
	network = "tcp"
	address = ":42069"
)

func main() {
	server := NewServer(address)
	server.Start()
}

type Server struct {
	address  string
	listener net.Listener
}

func NewServer(addr string) *Server {
	return &Server{
		address: addr,
	}
}

func (s *Server) Start() {
	listener, err := net.Listen(network, s.address)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	s.listener = listener
	defer s.listener.Close()

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Println("Shutting down server...")
		s.listener.Close()
	}()

	log.Printf("Server listening on %s\n", s.address)
	s.acceptConnections()
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		// Handle each connection in a goroutine
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		log.Printf("Connection to %v closed", conn.RemoteAddr())
	}()

	r, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	fmt.Printf("Request Line:\n - Method: %v\n - Target: %v\n - Version: %v\n",
		r.RequestLine.Method,
		r.RequestLine.RequestTarget,
		r.RequestLine.HttpVersion)

	fmt.Printf("Headers:\n")
	for k, v := range r.Headers {
		fmt.Printf(" - %v: %v\n", k, v)
	}

	fmt.Printf("Body:\n %v\n", string(r.Body))

}
