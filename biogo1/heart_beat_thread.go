package main

import (
	"fmt"
	"time"
)

type HeartBeatThread struct {
	lobbyServer       *ServerThread
	packetHandler     *PacketHandler
	gameServer  	  *GameServerThread
	gamePacketHandler *GameServerPacketHandler
}

func NewHeartBeatThread(lobbyServer *ServerThread, packetHandler *PacketHandler, gameServer *GameServerThread, gamePacketHandler *GameServerPacketHandler) *HeartBeatThread {
	return &HeartBeatThread{
		lobbyServer:       lobbyServer,
		packetHandler:     packetHandler,
		gameServer:        gameServer,
		gamePacketHandler: gamePacketHandler,
	}
}	

func (h *HeartBeatThread) Run() {
	for {
		fmt.Println("Heartbeat check running")
		time.Sleep(10 * time.Second) // Simulate keepalive ping
	}
}
