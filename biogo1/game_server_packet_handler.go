package main

import (
	"fmt"
	"net"
)

type GameServerPacketHandler struct{}

func (gsp *GameServerPacketHandler) Run() {
	fmt.Println("GameServerPacketHandler started")
	// Implement game packet handling logic here
}

// ProcessData is a stub function that prints debug information about incoming packets.
func (gsp *GameServerPacketHandler) ProcessData(server *GameServerThread, conn net.Conn, data []byte, length int) {
	fmt.Printf("Processing data from %s: %d bytes\n", conn.RemoteAddr(), length)
	fmt.Printf("Raw Data: %x\n", data) // Print raw bytes in hex format
}

func (gsp *GameServerPacketHandler) CountInGamePlayers() int {
	// TODO
	return 0
}