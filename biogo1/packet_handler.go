package main

import (
	"fmt"
	"main/commands"
	"net"
)

const (
	STATUS_OFFLINE = 0
	STATUS_LOBBY   = 1
	STATUS_GAME    = 2
	STATUS_AGLOBBY = 3
)

type PacketHandler struct {
	gameServerPacketHandler *GameServerPacketHandler
	packetIDCounter         int
	queue                   chan ServerDataEvent
	gameNumber              int
	db                      *Database
	clients                 *ClientList
}

func NewPacketHandler() *PacketHandler {
	return &PacketHandler{
		gameServerPacketHandler: nil,
		packetIDCounter:         0,
		queue:                   make(chan ServerDataEvent, 100),
		gameNumber:              1,
		clients:                 NewClientList(),
	}
}

func (ph *PacketHandler) Run() {
	// // Get configuration
	// conf := NewConfiguration() // Assuming a function to get config

	// // Resolve gameserver IP
	// tmpIP, err := net.ResolveIPAddr("ip", conf.gsIP)
	// if err != nil {
	// 	fmt.Println("Unknown Host, check properties file!")
	// } else {
	// 	ph.gsIP = tmpIP.IP
	// 	fmt.Println("Gameserver IP:", ph.gsIP)
	// }

	// // Open database connection
	db, err := NewDatabase("bioserver", "xxxxxxxxxxxxxxxx")
	if err != nil {
		fmt.Println("PacketHandler Run() Error opening database connection:", err)
		return
	}
	ph.db = db

	// // Initialize counters
	ph.packetIDCounter = 0
	ph.gameNumber = 1

	// // Setup patch
	// ph.patch = NewPatch()

	// // Setup areas, rooms, slots
	// ph.areas = NewAreas()
	// ph.rooms = NewRooms(ph.areas.GetAreaCount())
	// ph.slots = NewSlots(ph.areas.GetAreaCount(), ph.rooms.GetRoomCount())

	// Process the queue in a loop
	// (this loops forever)
	for event := range ph.queue {
		event.server.Send(event.socket, event.data)
	}
}

func (p *PacketHandler) SetGameServerPacketHandler(handler *GameServerPacketHandler) {
	p.gameServerPacketHandler = handler
}

func (p *PacketHandler) SendLogin(st *ServerThread, sc net.Conn) {
	// after connection the server sends its first packet, client answers
	seed := []byte{0x28, 0x37}
	pk := NewPacket(commands.LOGIN, commands.QUERY, commands.SERVER, p.getNextPacketID(), seed)
	p.addOutPacket(st, sc, pk)
}

// increase the server packet id
func (p *PacketHandler) getNextPacketID() int {
	p.packetIDCounter++
	return p.packetIDCounter
}

func (p *PacketHandler) addOutPacket(server *ServerThread, socket net.Conn, packet *Packet) {
	event := ServerDataEvent{
		server: server,
		socket: socket,
		data:   packet.GetPacketData(),
	}
	select {
	case p.queue <- event:
	default:
		fmt.Println("PacketHandler addOutPacket() PacketHandler queue full")
	}

}

func (ph *PacketHandler) ProcessData(server *ServerThread, socket net.Conn, data []byte) {
	offset := 0
	remaining := len(data)

	for remaining > 0 {
		if remaining < HEADER_SIZE {
			fmt.Printf("PacketHandler ProcessData() Incomplete packet: %d bytes remaining\n", remaining)
			break // or handle error
		}

		// Peek into header to get payload length
		// Assumes payload length is stored in data[4] and data[5]
		pLen := int(data[offset+4])<<8 | int(data[offset+5])
		packetSize := HEADER_SIZE + pLen
		if remaining < packetSize {
			fmt.Printf("PacketHandler ProcessData() Incomplete packet: %d bytes remaining\n", remaining)
			break // incomplete packet; wait for more data
		}
		packetData := data[offset : offset+packetSize]
		p := NewPacketFromBytes(packetData)
		

        fmt.Printf("PacketHandler ProcessData() - In cmd: 0x%X\n", p.cmd)
		if p.cmd == commands.LOGIN && ph.clients.FindClientBySocket(socket) != nil {
			fmt.Println("PacketHandler ProcessData() Dropping duplicate login packet")
		} else {
            ph.HandleInPacket(server, socket, p)
        }

		offset += packetSize
		remaining -= packetSize

        fmt.Printf("PacketHandler ProcessData() %p: Processed packet of size %d, remaining: %d\n", socket, packetSize, remaining)
	}

}

func (ph *PacketHandler) HandleInPacket(server *ServerThread, socket net.Conn, packet *Packet) {
	switch packet.who {
	case commands.CLIENT:
		switch packet.qsw {
		case commands.QUERY:
			switch packet.cmd {
			// case commands.UNKN61A0:
			//     send61A0(server, socket, packet)
			case commands.CHECKRND:
				ph.sendCheckRnd(server, socket, packet)
			// case commands.UNKN61A1:
			//     send61A1(server, socket, packet)
			// case commands.HNSELECT:
			//     sendHNSelect(server, socket, packet)
			// case commands.MOTHEDAY:
			//     sendMotheday(server, socket, packet)
			// case commands.CHARSELECT:
			//     sendCharSelect(server, socket, packet)
			// case commands.UNKN6881:
			//     send6881(server, socket, packet)
			// case commands.UNKN6882:
			//     send6882(server, socket, packet)
			// case commands.RANKINGS:
			//     sendRankings(server, socket, packet)
			// case commands.AREACOUNT:
			//     sendAreaCount(server, socket, packet)
			// case commands.AREAPLAYERCNT:
			//     sendAreaPlayerCnt(server, socket, packet)
			// case commands.AREASTATUS:
			//     sendAreaStatus(server, socket, packet)
			// case commands.AREANAME:
			//     sendAreaName(server, socket, packet)
			// case commands.AREADESCRIPT:
			//     sendAreaDescript(server, socket, packet)
			// case commands.AREASELECT:
			//     sendAreaSelect(server, socket, packet)
			// case commands.ROOMSCOUNT:
			//     sendRoomsCount(server, socket, packet)
			// case commands.ROOMPLAYERCNT:
			//     sendRoomPlayerCnt(server, socket, packet)
			// case commands.ROOMSTATUS:
			//     sendRoomStatus(server, socket, packet)
			// case commands.ROOMNAME:
			//     sendRoomName(server, socket, packet)
			// case commands.UNKN6308:
			//     send6308(server, socket, packet)
			// case commands.ENTERROOM:
			//     sendEnterRoom(server, socket, packet)
			default:
				fmt.Printf("PacketHandler HandleInPacket() Unknown command on query: %d (0x%X)\n", packet.cmd, packet.cmd)
			}
		case commands.TELL:
			switch packet.cmd {
			// case commands.CONNCHECK:
			//     cl := ph.clients.FindClient(socket)
			//     if cl != nil {
			//         cl.ConnAlive = true
			//     }
			case commands.LOGIN:
				fmt.Printf("PacketHandler HandleInPacket() Login packet received: %v\n", packet)
				if ph.checkSession(server, socket, packet) {
					fmt.Printf("PacketHandler HandleInPacket() Session check passed!\n")
					// correct session established
					// next step is the version check for File#1 updates
					ph.sendVersionCheck(server, socket)
				} else {
					fmt.Println("PacketHandler HandleInPacket() Session check failed!")
				}
			case commands.CHECKVERSION:
				if ph.checkPatchLevel(server, socket, packet) {
					// if version is older than actual patch, send patch
					fmt.Println("PacketHandler HandleInPacket() literally never reaches here....")
					// ph.beginPatch(server, socket)
				} else {
					// next step is to offer the registered handle/name pairs
					ph.sendIDHNPairs(server, socket)
				}
			// case commands.PATCHLINECHECK:
			//     continuePatch(server, socket, packet)
			// case commands.PATCHFINISH:
			//     sendShutdown(server, socket)
			default:
				fmt.Printf("PacketHandler HandleInPacket() Unknown command on answer: %d (0x%X)\n", packet.cmd, packet.cmd)
			}
		case commands.BROADCAST:
			switch packet.cmd {
			// case commands.STARTGAME:
			//     broadcastGetReady(server, socket)
			// case commands.CHATIN:
			//     broadcastChatOut(server, socket, packet)
			default:
				fmt.Printf("PacketHandler HandleInPacket() Unknown command on broadcast: %d (0x%X)\n", packet.cmd, packet.cmd)
			}
		default:
			fmt.Printf("PacketHandler HandleInPacket() Unknown qsw type on incoming packet! 0x%X\n", packet.qsw)
		}
	default:
		fmt.Println("PacketHandler HandleInPacket() Not a client who on incoming packet!")
	}
}

func (ph *PacketHandler) checkSession(server *ServerThread, socket net.Conn, p *Packet) bool {
	// this should probably be renamed or broken into different functions
	// since it does more than just check the session
	seed := p.pid

	sessA := int(p.pay[2]-0x30) * 10000
	sessA += int(p.pay[3]-0x30) * 1000
	sessA += int(p.pay[4]-0x30) * 100
	sessA += int(p.pay[5]-0x30) * 10
	sessA += int(p.pay[6] - 0x30)

	sessB := int(p.pay[7]-0x30) * 10000
	sessB += int(p.pay[8]-0x30) * 1000
	sessB += int(p.pay[9]-0x30) * 100
	sessB += int(p.pay[10]-0x30) * 10
	sessB += int(p.pay[11] - 0x30)

	session := fmt.Sprintf("%04d%04d", sessA-seed, sessB-seed)

	userid, err := ph.db.GetUserID(session)
	if err != nil {
		fmt.Println("PacketHandler checkSession() Error getting user id:", err)
		return false
	}

	fmt.Printf("PacketHandler checkSession() Session: %s with UserID: %s\n", session, userid)


	if userid != "" {
		// loop through clients and remove old connections
		// then setup client object for this user/session
		// TODO: (should this be implemented by clients.go instead?)
		cl := ph.clients.FindClientByUserID(userid)
		if cl != nil {
            fmt.Printf("PacketHandler checkSession() Found a client with UserID %s: %v\n", userid, cl)
			fmt.Println("PacketHandler checkSession() TODO looping through clients to remove old connections...")
			// for _, c := range ph.clients.GetList() {
			// 	if c.userID == userid {
            //         fmt.Printf("We want to remove client with UserID %s: %v\n", userid, c)
            //         // server.Disconnect(c.socket)

			// 		// ph.removeClient(server, c)

            //         // pretty sure this one is a dupe:
			// 		// server.Disconnect(c.socket)
			// 	}
			// }
		}

		ph.clients.Add(NewClient(socket, userid, session))
		cl = ph.clients.FindClientBySocket(socket)
		if cl == nil {
			fmt.Println("HandleInPacket checkSession() Failed to add client to client list!!! big problem!!!")
			return false
		}

		err = ph.db.UpdateClientOrigin(userid, STATUS_LOBBY, 0, 0, 0)
		if err != nil {
			fmt.Println("HandleInPacket checkSession() Error updating client origin:", err)
			return false
		}

		gamenr, err := ph.db.GetGameNumber(cl.userID)
		if err != nil {
			fmt.Println("HandleInPacket checkSession() Error getting game number:", err)
			return false
		}
		if gamenr > 0 {
			// we are in meeting room then
			// game number not set yet because needed for broadcast packets in AGL!
			cl.area = 51
			ph.db.UpdateClientOrigin(userid, STATUS_AGLOBBY, 51, 0, 0)
		}
		return true
	} else {
		// session check failed; disconnect this client
		fmt.Println("HandleInPacket checkSession() Session check failed!")
		return false
	}
}

func (ph *PacketHandler) sendVersionCheck(server *ServerThread, socket net.Conn) {
	pk := NewPacket(commands.CHECKVERSION, commands.QUERY, commands.SERVER, ph.getNextPacketID(), []byte{0x00, 0x00})
	ph.addOutPacket(server, socket, pk)
}

func (ph *PacketHandler) sendCheckRnd(server *ServerThread, socket net.Conn, p *Packet) {
	teststring := []byte{0x00, 0x01, 0x30}
	p.CryptString()
	teststring[2] = p.pay[4]
	pk := NewPacket(commands.CHECKRND, commands.TELL, commands.SERVER, p.pid, teststring)
	ph.addOutPacket(server, socket, pk)
}

func (ph *PacketHandler) checkPatchLevel(server *ServerThread, socket net.Conn, p *Packet) bool {
	// for testing just get the version string and dump the packet

	// packetData := p.GetPacketData()
	// fmt.Printf("Packet data: %v\n", packetData)
	// version := p.GetVersion()
	// fmt.Printf("Decrypted client version: %s\n", version)

	// check if the client has the latest patch level
	// if not, send patch
	return false
}

func (ph *PacketHandler) sendIDHNPairs(server *ServerThread, socket net.Conn) {
	// get the handles tied to this userid (max 3)
	userid := ph.clients.FindClientBySocket(socket).userID

	hn := ph.db.GetHNPairs(userid)

	fmt.Printf("PacketHandler sendIDHNPairs() Sending HNPairs: %v\n", hn)

	pk := NewPacket(commands.IDHNPAIRS, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), hn.GetArray())
	ph.addOutPacket(server, socket, pk)

}

func (ph *PacketHandler) removeClient(server *ServerThread, cl *Client) {
	if cl == nil {
		return
	}
	fmt.Printf("Removing client %s\n", cl.userID)
	// // If needed, lock the client (e.g., cl.mu.Lock(); defer cl.mu.Unlock())
	// area := cl.area
	// room := cl.room
	// slot := cl.slot
	// game := cl.gamenumber
	socket := cl.socket
	// host := cl.host
	// who := cl.GetHNPair().handle

	// Set the client status to offline.
	if err := ph.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0); err != nil {
		fmt.Println("Error updating client origin to offline:", err)
	}

	// Remove the client from the list.
	ph.clients.Remove(cl)

	// // If the client was a host and occupying a slot, perform slot-specific broadcasts.
	// if host == 1 && slot != 0 {
	// 	ph.slots.GetSlot(area, room, slot).Reset()
	// 	ph.broadcastCancelSlot(server, area, room, slot)
	// 	ph.broadcastPasswdProtect(server, area, room, slot)
	// 	ph.broadcastSlotSceneType(server, area, room, slot)
	// 	ph.broadcastSlotTitle(server, area, room, slot)
	// 	ph.broadcastSlotAttrib2(server, area, room, slot)
	// 	ph.broadcastSlotPlayerStatus(server, area, room, slot)
	// 	ph.broadcastSlotStatus(server, area, room, slot)
	// }

	// // If the client was not a host but still in a slot.
	// if slot != 0 && host == 0 {
	// 	// Prepare a broadcast packet to notify other players in the slot.
	// 	wholeaves := []byte{0, 6, 0, 0, 0, 0, 0, 0}
	// 	copy(wholeaves[2:], who)
	// 	p := NewPacket(Commands.LEAVESLOT, Commands.BROADCAST, Commands.SERVER, ph.getNextPacketID(), wholeaves)
	// 	ph.broadcastInSlot(server, p, area, room, slot)

	// 	// If there is room for additional players and a host is still present, update slot status.
	// 	n := ph.clients.CountPlayersInSlot(area, room, slot)
	// 	maxPlayers := ph.slots.GetMaximumPlayers(area, room, slot)
	// 	if n < maxPlayers {
	// 		if ph.clients.GetHostOfSlot(area, room, slot) != nil {
	// 			ph.slots.GetSlot(area, room, slot).SetStatus(SlotStatusGameSet)
	// 		}
	// 	}

	// 	// If this was the last client in the slot, reset the slot and broadcast related changes.
	// 	if ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
	// 		ph.slots.GetSlot(area, room, slot).Reset()
	// 		ph.broadcastPasswdProtect(server, area, room, slot)
	// 		ph.broadcastSlotSceneType(server, area, room, slot)
	// 		ph.broadcastSlotTitle(server, area, room, slot)
	// 	}

	// 	ph.broadcastSlotAttrib2(server, area, room, slot)
	// 	ph.broadcastSlotPlayerStatus(server, area, room, slot)
	// 	ph.broadcastSlotStatus(server, area, room, slot)
	// }

	// // In the after-game lobby (area 51) with a valid game number, you might need extra handling.
	// if area == 51 && game != 0 {
	// 	// TODO: is this really necessary?
	// }

	// // Broadcast the updated room player count.
	// ph.broadcastRoomPlayerCnt(server, area, room)

	// Finally, disconnect the client socket.
	server.Disconnect(socket)
}

func (ph *PacketHandler) RemoveClientNoDisconnect(server *ServerThread, socket net.Conn) {
    cl := ph.clients.FindClientBySocket(socket)

	if cl == nil {
		return
	}
    cl.ConnAlive = false

	fmt.Printf("PacketHandler Removing client no disconnect %s\n", cl.userID)
	// // If needed, lock the client (e.g., cl.mu.Lock(); defer cl.mu.Unlock())
	// area := cl.area
	// room := cl.room
	// slot := cl.slot
	// game := cl.gamenumber
	// socket := cl.socket
	// host := cl.host
	// who := cl.GetHNPair().handle

	// Set the client status to offline.
	if err := ph.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0); err != nil {
		fmt.Println("PacketHandler RemoveClientNoDisconnect() Error updating client origin to offline:", err)
	}

	// Remove the client from the list.
	ph.clients.Remove(cl)
	fmt.Printf("PacketHandler RemoveClientNoDisconnect() Client %s removed but kept session alive\n", cl.userID)

	// // If the client was a host and occupying a slot, perform slot-specific broadcasts.
	// if host == 1 && slot != 0 {
	// 	ph.slots.GetSlot(area, room, slot).Reset()
	// 	ph.broadcastCancelSlot(server, area, room, slot)
	// 	ph.broadcastPasswdProtect(server, area, room, slot)
	// 	ph.broadcastSlotSceneType(server, area, room, slot)
	// 	ph.broadcastSlotTitle(server, area, room, slot)
	// 	ph.broadcastSlotAttrib2(server, area, room, slot)
	// 	ph.broadcastSlotPlayerStatus(server, area, room, slot)
	// 	ph.broadcastSlotStatus(server, area, room, slot)
	// }

	// // If the client was not a host but still in a slot.
	// if slot != 0 && host == 0 {
	// 	// Prepare a broadcast packet to notify other players in the slot.
	// 	wholeaves := []byte{0, 6, 0, 0, 0, 0, 0, 0}
	// 	copy(wholeaves[2:], who)
	// 	p := NewPacket(Commands.LEAVESLOT, Commands.BROADCAST, Commands.SERVER, ph.getNextPacketID(), wholeaves)
	// 	ph.broadcastInSlot(server, p, area, room, slot)

	// 	// If there is room for additional players and a host is still present, update slot status.
	// 	n := ph.clients.CountPlayersInSlot(area, room, slot)
	// 	maxPlayers := ph.slots.GetMaximumPlayers(area, room, slot)
	// 	if n < maxPlayers {
	// 		if ph.clients.GetHostOfSlot(area, room, slot) != nil {
	// 			ph.slots.GetSlot(area, room, slot).SetStatus(SlotStatusGameSet)
	// 		}
	// 	}

	// 	// If this was the last client in the slot, reset the slot and broadcast related changes.
	// 	if ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
	// 		ph.slots.GetSlot(area, room, slot).Reset()
	// 		ph.broadcastPasswdProtect(server, area, room, slot)
	// 		ph.broadcastSlotSceneType(server, area, room, slot)
	// 		ph.broadcastSlotTitle(server, area, room, slot)
	// 	}

	// 	ph.broadcastSlotAttrib2(server, area, room, slot)
	// 	ph.broadcastSlotPlayerStatus(server, area, room, slot)
	// 	ph.broadcastSlotStatus(server, area, room, slot)
	// }

	// // In the after-game lobby (area 51) with a valid game number, you might need extra handling.
	// if area == 51 && game != 0 {
	// 	// TODO: is this really necessary?
	// }

	// // Broadcast the updated room player count.
	// ph.broadcastRoomPlayerCnt(server, area, room)
}