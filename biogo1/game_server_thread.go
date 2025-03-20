package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type GameServerThread struct {
	hostAddress    string
	port           int
	packetHandler  *GameServerPacketHandler
	listener       net.Listener
	changeRequests chan ServerChangeEvent
	pendingData    map[net.Conn][][]byte
	readBuffers    map[net.Conn]*ServerStreamBuffer
	mu             sync.Mutex
	initOK         bool
}

func NewGameServerThread(hostAddress string, port int, packetHandler *GameServerPacketHandler) *GameServerThread {
	log.Printf("Initializing GameServer at %s:%d\n", hostAddress, port)
	return &GameServerThread{
		hostAddress:    hostAddress,
		port:           port,
		packetHandler:  packetHandler,
		changeRequests: make(chan ServerChangeEvent, 100),
		pendingData:    make(map[net.Conn][][]byte),
		readBuffers:    make(map[net.Conn]*ServerStreamBuffer),
		initOK:         true,
	}
}

func (g *GameServerThread) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	addr := fmt.Sprintf("%s:%d", g.hostAddress, g.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Error starting server on %s: %v", addr, err)
	}
	g.listener = ln
	fmt.Printf("Game server started on port %d\n", g.port)
	go g.processChangeRequests()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		g.accept(conn)
	}
}

func (g *GameServerThread) processChangeRequests() {
	for change := range g.changeRequests {
		switch change.EventType {
		case CHANGEOPS:
			// go does not need to manage interestOps
		case FORCECLOSE:
			g.close(change.conn)
		}
	}
}

func (g *GameServerThread) accept(conn net.Conn) {
	fmt.Println("New game connection from", conn.RemoteAddr())
	g.mu.Lock()
	if _, exists := g.readBuffers[conn]; !exists {
		g.readBuffers[conn] = NewServerStreamBuffer()
	}
	g.mu.Unlock()
	if g.packetHandler != nil {
		// g.packetHandler.GSsendLogin(g, conn)
	}
	go g.read(conn)
}

func (g *GameServerThread) read(conn net.Conn) {
	defer g.close(conn)
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("game Error reading from", conn.RemoteAddr(), err)
			return
		}
		if n == 0 {
			continue
		}
		data := buffer[:n]
		// acquires a lock to check if there's already a ServerStreamBuffer
		// for the connection, if not, creates a new one
		// and stores it in g.readBuffers[conn]
		g.mu.Lock()
		rb, ok := g.readBuffers[conn]
		g.mu.Unlock()
		if !ok {
			rb = NewServerStreamBuffer()
			g.mu.Lock()
			g.readBuffers[conn] = rb
			g.mu.Unlock()
		}

		msg := rb.Append(data)

		// if rb.Append returns a complete message (ie msg != nil)
		// we then call g.packetHandler.processData to handle the message
		if msg != nil && g.packetHandler != nil {
			g.packetHandler.ProcessData(g, conn, msg, len(msg))
		}
	}
}

func (g *GameServerThread) write(conn net.Conn) {
	for {
		g.mu.Lock()
		queue, exists := g.pendingData[conn]
		if !exists || len(queue) == 0 {
			g.mu.Unlock()
			break
		}
		data := queue[0]
		g.mu.Unlock()
		n, err := conn.Write(data)
		if err != nil {
			log.Println("game Error writing to", conn.RemoteAddr(), err)
			g.close(conn)
			return
		}
		if n < len(data) {
			g.mu.Lock()
			g.pendingData[conn][0] = data[n:]
			g.mu.Unlock()
		} else {
			g.mu.Lock()
			g.pendingData[conn] = g.pendingData[conn][1:]
			g.mu.Unlock()
		}
	}
}

func (g *GameServerThread) send(conn net.Conn, data []byte) {
	g.mu.Lock()
	g.pendingData[conn] = append(g.pendingData[conn], data)
	g.mu.Unlock()
	go g.write(conn)
}

func (g *GameServerThread) disconnect(conn net.Conn) {
	g.changeRequests <- ServerChangeEvent{conn: conn, EventType: FORCECLOSE, ops: 0}
}

func (g *GameServerThread) close(conn net.Conn) {
	conn.Close()
	g.mu.Lock()
	delete(g.readBuffers, conn)
	delete(g.pendingData, conn)
	g.mu.Unlock()
	if g.packetHandler != nil {
		// ph.debug("game Removing client %s\n", conn.RemoteAddr())
		// g.packetHandler.removeClientNoDisconnect(g, conn)
		return
	}
}
