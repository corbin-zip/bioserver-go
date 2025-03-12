package main

import(
	"fmt"
)

type PacketHandler struct {
	gameServerPacketHandler *GameServerPacketHandler
}

func (p *PacketHandler) Run() {
	fmt.Println("PacketHandler started")
}

func (p *PacketHandler) SetGameServerPacketHandler(handler *GameServerPacketHandler) {
	p.gameServerPacketHandler = handler
}