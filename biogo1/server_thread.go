package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type ServerThread struct {
	addr          *net.TCPAddr
	packetHandler *PacketHandler
}

func NewServerThread(address string, port int, packetHandler *PacketHandler) (*ServerThread, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %s:%d: %w", address, port, err)
	}
	log.Printf("Initializing server at %s\n", tcpAddr.String())
	return &ServerThread{addr: tcpAddr, packetHandler: packetHandler}, nil
}

func (s *ServerThread) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.ListenTCP("tcp", s.addr)
	if err != nil {
		log.Fatalf("Error starting server on %s: %v", s.addr.String(), err)
	}
	fmt.Printf("Lobby server started on %s\n", s.addr.String())
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *ServerThread) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("New connection from", conn.RemoteAddr())
	// if s.packetHandler != nil {
	// 	s.packetHandler.Handle(conn)
	// }
}
