package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type GameServerThread struct {
	address       string
	port          int
	packetHandler *GameServerPacketHandler
}

func NewGameServerThread(address string, port int, packetHandler *GameServerPacketHandler) *GameServerThread {
	log.Printf("Initializing GameServer at %s:%d\n", address, port)
	return &GameServerThread{
		address:       address,
		port:          port,
		packetHandler: packetHandler,
	}
}

func (g *GameServerThread) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", g.port))
	if err != nil {
		log.Fatalf("Error starting server on port %d: %v", g.port, err)
	}
	fmt.Printf("Game server started on port %d\n", g.port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go g.handleConnection(conn)
	}
}

func (g *GameServerThread) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("New game connection from", conn.RemoteAddr())
	// if g.packetHandler != nil {
	// 	g.packetHandler.Handle(conn)
	// }
}
