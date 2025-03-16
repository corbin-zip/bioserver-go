package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type ServerThread struct {
	addr           *net.TCPAddr
	packetHandler  *PacketHandler
	listener       *net.TCPListener
	changeRequests chan ServerChangeEvent
	pendingData    map[net.Conn][][]byte
	readBuffers    map[net.Conn]*ServerStreamBuffer
	initOK         bool
	mu             sync.Mutex
}

func NewServerThread(address string, port int, packetHandler *PacketHandler) (*ServerThread, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", address, port))

	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %s:%d: %w", address, port, err)
	}

	log.Printf("Initializing Server at %s\n", tcpAddr.String())

	return &ServerThread{
		addr:           tcpAddr,
		packetHandler:  packetHandler,
		changeRequests: make(chan ServerChangeEvent, 100),
		pendingData:    make(map[net.Conn][][]byte),
		readBuffers:    make(map[net.Conn]*ServerStreamBuffer),
		initOK:         true,
	}, nil
}

func (s *ServerThread) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	ln, err := net.ListenTCP("tcp", s.addr)
	if err != nil {
		log.Fatalf("Error starting server on %s: %v", s.addr.String(), err)
	}
	s.listener = ln
	fmt.Printf("Lobby server started on %s\n", s.addr.String())
	go s.processChangeRequests()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go s.accept(conn)
	}
}

func (s *ServerThread) processChangeRequests() {
	for change := range s.changeRequests {
		switch change.EventType {
		case CHANGEOPS:
			// go does not need to manage interestOps
		case FORCECLOSE:
			// close the connection
			// change.Conn.Close()
			s.close(change.conn)
		}
	}
}

func (s *ServerThread) accept(conn net.Conn) {
	fmt.Println("New connection from", conn.RemoteAddr())
	s.mu.Lock()
	if _, exists := s.readBuffers[conn]; !exists {
		s.readBuffers[conn] = NewServerStreamBuffer()
	}
	s.mu.Unlock()
	if s.packetHandler != nil {
		fmt.Printf("%p conn SendLogin() to it\n", conn)
		s.packetHandler.SendLogin(s, conn)
	}
	go s.read(conn)
}

func (s *ServerThread) read(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("Read error from", conn.RemoteAddr(), err)
			s.mu.Lock()
			delete(s.readBuffers, conn)
			s.mu.Unlock()
			s.close(conn)
			return
		}
		if n == 0 {
			continue
		}
		data := buffer[:n]
		s.mu.Lock()
		rb, exists := s.readBuffers[conn]
		if !exists {
			rb = NewServerStreamBuffer()
			s.readBuffers[conn] = rb
		}
		s.mu.Unlock()
		if err := rb.AppendData(data); err != nil {
			log.Println("Buffer overflow for", conn.RemoteAddr())
			s.close(conn)
			return
		}
		msg := rb.GetCompleteMessages()
		if msg != nil && s.packetHandler != nil {
			s.packetHandler.ProcessData(s, conn, msg)
		}
	}
}

func (s *ServerThread) write(conn net.Conn) {
	for {
		s.mu.Lock()
		queue, exists := s.pendingData[conn]
		if !exists || len(queue) == 0 {
			s.mu.Unlock()
			break
		}
		data := queue[0]
		s.mu.Unlock()

		n, err := conn.Write(data)
		if err != nil {
			log.Println("Write error to", conn.RemoteAddr(), err)
			s.close(conn)
			return
		}

		// if n < len(data) {
		// 	s.mu.Lock()
		// 	s.pendingData[conn][0] = data[n:]
		// 	s.mu.Unlock()
		// } else {
		// 	s.mu.Lock()
		// 	s.pendingData[conn] = s.pendingData[conn][1:]
		// 	s.mu.Unlock()
		// }
		s.mu.Lock()
		if n < len(data) {
			s.pendingData[conn][0] = data[n:]
		} else {
			if len(s.pendingData[conn]) > 1 { // Prevent slice bounds out of range
				s.pendingData[conn] = s.pendingData[conn][1:]
			} else {
				delete(s.pendingData, conn) // Ensure it's removed when empty
			}
		}
		s.mu.Unlock()
	}
}

func (s *ServerThread) Send(conn net.Conn, data []byte) {
	s.mu.Lock()
	s.pendingData[conn] = append(s.pendingData[conn], data)
	s.mu.Unlock()
	go s.write(conn)
}

func (s *ServerThread) Disconnect(conn net.Conn) {
	s.changeRequests <- ServerChangeEvent{conn: conn, EventType: FORCECLOSE, ops: 0}
}

func (s *ServerThread) close(conn net.Conn) {
	conn.Close()
	s.mu.Lock()
	delete(s.readBuffers, conn)
	delete(s.pendingData, conn)
	s.mu.Unlock()
	if s.packetHandler != nil {
		fmt.Printf("Trying to remove client no disconnect %s\n", conn.RemoteAddr())
		s.packetHandler.RemoveClientNoDisconnect(s, conn)
	}
}
