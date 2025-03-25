package main

import (
	"time"
)

type HeartBeatThread struct {
	lobbyServer       *ServerThread
	packetHandler     *PacketHandler
	gameServer        *GameServerThread
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

func (hbt *HeartBeatThread) Run() {
	counter := 0
	counter2 := 0

	for {
		hbt.packetHandler.BroadcastPing(hbt.lobbyServer)
		hbt.gamePacketHandler.ConnCheck(hbt.gameServer)
		hbt.packetHandler.CheckAutoStart(hbt.lobbyServer)
		if counter == 1 {
			hbt.packetHandler.BroadcastConnCheck(hbt.lobbyServer)
			counter = 0
		} else {
			counter++
		}

		if counter2 == 9 {
			hbt.packetHandler.CleanGhostRooms(hbt.lobbyServer)
			counter2 = 0
		} else {
			counter2++
		}

		time.Sleep(30 * time.Second) // Simulate keepalive ping
	}
}
