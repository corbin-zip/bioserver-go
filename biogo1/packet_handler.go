package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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
	areas                   *Areas
	rooms                   *Rooms
	slots                   *Slots
	// rules                   *RuleSet
}

func NewPacketHandler() *PacketHandler {
	ph := &PacketHandler{}
	ph.gameServerPacketHandler = nil
	ph.packetIDCounter = 0
	ph.queue = make(chan ServerDataEvent, 100)
	ph.gameNumber = 1
	ph.clients = NewClientList()
	ph.areas = NewAreas()
	ph.rooms = NewRooms(ph.areas.GetAreaCount())
	ph.slots = NewSlots(ph.areas.GetAreaCount(), ph.rooms.GetRoomCount())
	// ph.rules = NewRuleSet()
	return ph
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

func (ph *PacketHandler) addOutPacket(server *ServerThread, socket net.Conn, p *Packet) {
	fmt.Printf("PacketHandler addOutPacket() 0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))

	event := ServerDataEvent{
		server: server,
		socket: socket,
		data:   p.GetPacketData(),
	}
	select {
	case ph.queue <- event:
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

		// fmt.Printf("PacketHandler ProcessData() - In cmd: 0x%X (%s)\n", p.cmd, commands.GetConstName(p.cmd))
		// if p.cmd == commands.LOGIN && ph.clients.FindClientBySocket(socket) != nil {
		// fmt.Println("PacketHandler ProcessData() Dropping duplicate login packet")
		// } else {
		ph.HandleInPacket(server, socket, p)
		// }

		offset += packetSize
		remaining -= packetSize

		// fmt.Printf("PacketHandler ProcessData() %p: Processed packet of size %d, remaining: %d\n", socket, packetSize, remaining)
	}

}

func (ph *PacketHandler) HandleInPacket(server *ServerThread, socket net.Conn, packet *Packet) {
	p := packet
	fmt.Printf("PacketHandler HandleInPacket() 0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))

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
			case commands.HNSELECT:
				ph.sendHNSelect(server, socket, packet)
			case commands.UNKN6002:
				ph.send6002(server, socket, packet)
			case commands.MOTHEDAY:
				ph.sendMotheday(server, socket, packet)
			case commands.CHARSELECT:
				ph.sendCharSelect(server, socket, packet)
			case commands.UNKN6881:
				ph.send6881(server, socket, packet)
			case commands.UNKN6882:
				ph.send6882(server, socket, packet)
			case commands.RANKINGS:
				ph.sendRankings(server, socket, packet)
			case commands.AREACOUNT:
				ph.sendAreaCount(server, socket, packet)
			case commands.AREAPLAYERCNT:
				ph.sendAreaPlayerCnt(server, socket, packet)
			case commands.AREASTATUS:
				ph.sendAreaStatus(server, socket, packet)
			case commands.AREANAME:
				ph.sendAreaName(server, socket, packet)
			case commands.AREADESCRIPT:
				ph.sendAreaDescript(server, socket, packet)
			case commands.AREASELECT:
				ph.sendAreaSelect(server, socket, packet)
			case commands.ROOMSCOUNT:
				ph.sendRoomsCount(server, socket, packet)
			case commands.ROOMPLAYERCNT:
				ph.sendRoomPlayerCnt(server, socket, packet)
			case commands.ROOMSTATUS:
				ph.sendRoomStatus(server, socket, packet)
			case commands.ROOMNAME:
				ph.sendRoomName(server, socket, packet)
			case commands.UNKN6308:
				ph.send6308(server, socket, packet)
			case commands.ENTERROOM:
				ph.sendEnterRoom(server, socket, packet)
			case commands.SLOTCOUNT:
				ph.sendSlotCount(server, socket, packet)
			case commands.SLOTSTATUS:
				ph.sendSlotStatus(server, socket, packet)
			case commands.SLOTPLRSTATUS:
				ph.sendSlotPlayerStatus(server, socket, packet)
			case commands.SLOTTITLE:
				ph.sendSlotTitle(server, socket, packet)
			case commands.SLOTATTRIB2:
				ph.sendSlotAttrib2(server, socket, packet)
			case commands.SLOTPWDPROT:
				ph.sendPasswdProtect(server, socket, packet)
			case commands.SLOTSCENTYPE:
				ph.sendSlotSceneType(server, socket, packet)
			case commands.RULESCOUNT:
				ph.sendRulesCount(server, socket, packet)
			case commands.RULEATTCOUNT:
				ph.sendRuleAttCount(server, socket, packet)
			case commands.UNKN6601:
				ph.send6601(server, socket, packet)
			case commands.UNKN6602:
				ph.send6602(server, socket, packet)
			case commands.RULEDESCRIPT:
				ph.sendRuleDescript(server, socket, packet)
			case commands.RULEVALUE:
				ph.sendRuleValue(server, socket, packet)
			case commands.RULEATTRIB:
				ph.sendRuleAttrib(server, socket, packet)
			// case commands.ATTRDESCRIPT:
			// 	ph.sendAttrDescript(server, socket, packet)
			// case commands.ATTRATTRIB:
			// 	ph.sendAttrAttrib(server, socket, packet)
			// case commands.PLAYERSTATS:
			// 	ph.sendPlayerStats(server, socket, packet)
			// case commands.EXITSLOTLIST:
			// 	ph.sendExitSlotlist(server, socket, packet)
			// case commands.EXITAREA:
			// 	ph.sendExitArea(server, socket, packet)
			case commands.CREATESLOT:
				ph.sendCreateSlot(server, socket, packet)
			// case commands.SCENESELECT:
			// 	ph.sendSceneSelect(server, socket, packet)
			// case commands.SLOTNAME:
			// 	ph.sendSlotName(server, socket, packet)
			// case commands.SETRULE:
			// 	ph.sendSetRule(server, socket, packet)
			// case 0x660c:
			// 	ph.send660c(server, socket, packet)
			// case commands.SLOTTIMER:
			// 	ph.sendSlotTimer(server, socket, packet)
			// case 0x6412:
			// 	ph.send6412(server, socket, packet)
			// case 0x6504:
			// 	ph.send6504(server, socket, packet)
			// case commands.CANCELSLOT:
			// 	ph.sendCancelSlot(server, socket, packet)
			// case commands.SLOTPASSWD:
			// 	ph.sendSlotPasswd(server, socket, packet)
			// case commands.PLAYERCOUNT:
			// 	ph.sendPlayerCount(server, socket, packet)
			// case commands.PLAYERNUMBER:
			// 	ph.sendPlayerNumber(server, socket, packet)
			// case commands.PLAYERSTAT:
			// 	ph.sendPlayerStat(server, socket, packet)
			// case commands.PLAYERSCORE:
			// 	ph.sendPlayerScore(server, socket, packet)
			// case commands.GAMESESSION:
			// 	ph.sendGameSession(server, socket, packet)
			// case commands.GAMEDIFF:
			// 	ph.sendDifficulty(server, socket, packet)
			// case commands.GSINFO:
			// 	ph.sendGSinfo(server, socket, packet)
			// case commands.ENTERAGL:
			// 	ph.sendEnterAGL(server, socket, packet)
			// case commands.AGLSTATS:
			// 	ph.sendAGLstats(server, socket, packet)
			// case commands.AGLPLAYERCNT:
			// 	ph.sendAGLplayerCnt(server, socket, packet)
			// case commands.LEAVEAGL:
			// 	ph.sendLeaveAGL(server, socket, packet)
			// case commands.JOINGAME:
			// 	ph.sendJoinGame(server, socket, packet)
			// case commands.GETINFO:
			// 	ph.sendGetInfo(server, socket, packet)
			// case commands.EVENTDAT:
			// 	ph.sendEventDat(server, socket, packet)
			// case commands.BUDDYLIST:
			// 	ph.sendBuddyList(server, socket, packet)
			// case commands.CHECKBUDDY:
			// 	ph.sendCheckBuddy(server, socket, packet)
			// case commands.PRIVATEMSG:
			// 	ph.sendPrivateMsg(server, socket, packet)
			// case commands.UNKN6181:
			// 	ph.send6181(server, socket, packet)
			// case commands.LOGOUT:
			// 	ph.sendLogout(server, socket, packet)
			default:
				fmt.Printf("PacketHandler HandleInPacket() Unknown or unimplemented command on query: 0x%X (%s)\n", packet.cmd, commands.GetConstName(packet.cmd))
			}
		case commands.TELL:
			switch packet.cmd {
			// case commands.CONNCHECK:
			//     cl := ph.clients.FindClient(socket)
			//     if cl != nil {
			//         cl.ConnAlive = true
			//     }
			case commands.LOGIN:
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
			for _, c := range ph.clients.GetList() {
				if c.userID == userid {
					fmt.Printf("PacketHandler checkSession() removing client with UserID %s & socket %p\n", userid, c.socket)
					ph.removeClient(server, c)
				}
			}
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

	packetData := p.GetPacketData()
	fmt.Printf("PacketHandler checkPatchLevel() Packet data: %v\n", packetData)
	version := p.GetVersion()
	fmt.Printf("PacketHandler checkPatchLevel() Decrypted client version: %s\n", version)

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

func (ph *PacketHandler) sendHNSelect(server *ServerThread, socket net.Conn, ps *Packet) {
	//TODO: optimize FindClient[...] calls; can we just pass in a client or no?
	var p *Packet
	chosen := make([]byte, 8)
	hn := ps.GetDecryptedHNPair()

	ph.clients.FindClientBySocket(socket).hnPair = hn

	if string(hn.handle) == "******" {
		hn.CreateHandle(ph.db)
		ph.db.CreateNewHNPair(ph.clients.FindClientBySocket(socket))
	}

	// update name of the handle in the database (why??)
	ph.db.UpdateHNPair(ph.clients.FindClientBySocket(socket))

	//send chosen handle as answer

	chosen[0] = 0
	chosen[1] = 6

	copy(chosen[2:8], hn.handle[:6])
	p = NewPacket(commands.HNSELECT, commands.TELL, commands.SERVER, ps.pid, chosen)
	ph.addOutPacket(server, socket, p)

	userid := ph.clients.FindClientBySocket(socket).userID
	gamenr, err := ph.db.GetGameNumber(userid)
	if err != nil {
		fmt.Printf("Error getting game number for userid %s: %v\n", userid, err)
	}

	// ask for info if user is coming from a game
	if gamenr > 0 {
		p = NewPacketWithoutPayload(commands.POSTGAMEINFO, commands.QUERY, commands.SERVER, ph.getNextPacketID())
		ph.addOutPacket(server, socket, p)
	}

	// end of the login procedure !!!!
	p = NewPacketWithoutPayload(commands.UNKN6104, commands.BROADCAST, commands.SERVER, ph.getNextPacketID())
	ph.addOutPacket(server, socket, p)

}

func (ph *PacketHandler) sendMotheday(server *ServerThread, socket net.Conn, p *Packet) {
	// message, err := ph.db.GetMOTD()
	// if err != nil {
	// 	message = "error getting motd..."
	// 	fmt.Printf("Failed to get MOTD: %v\n", err)
	// }
	// TODO testing
	message := "nomotd"
	fmt.Printf("PacketHandler sendMotheday() sending MOTD message: %s\n", message)
	motd := NewMOTD(1, message)
	motdp := NewPacket(commands.MOTHEDAY, commands.TELL, commands.SERVER, p.pid, motd.GetPacket())
	ph.addOutPacket(server, socket, motdp)
}

func (ph *PacketHandler) sendCharSelect(server *ServerThread, socket net.Conn, p *Packet) {
	// cl := ph.clients.FindClientBySocket(socket)
	// cl.SetCharacterStats(p.GetCharacterStats())

	outp := NewPacketWithoutPayload(commands.CHARSELECT, commands.TELL, commands.SERVER, p.pid)
	ph.addOutPacket(server, socket, outp)
}

func (ph *PacketHandler) send6881(server *ServerThread, socket net.Conn, p *Packet) {
	datacount := []byte{0x01, 0, 0, 0x12, 0x5D}

	outp := NewPacket(commands.UNKN6881, commands.TELL, commands.SERVER, p.pid, datacount)
	ph.addOutPacket(server, socket, outp)
}

func (ph *PacketHandler) send6882(server *ServerThread, socket net.Conn, p *Packet) {
	pl := p.pay // []byte containing the packet payload
	nr := int(pl[0])
	offset := int(pl[1])<<24 | int(pl[2])<<16 | int(pl[3])<<8 | int(pl[4])
	sizeL := int(pl[5])<<24 | int(pl[6])<<16 | int(pl[7])<<8 | int(pl[8])
	data := Packet6881GetData(nr, offset, sizeL)
	outp := NewPacket(commands.UNKN6882, commands.TELL, commands.SERVER, p.pid, data)
	ph.addOutPacket(server, socket, outp)
}

// sendRankings requests player rankings per area.
// For the moment we are sending the same (empty) rankings for every area.
// Format:
// areanumber, x1, x2, x3, x4, points rank 7204th, cleartime rank 16998th, 13500 points, x8, x9, x10
// status(1 alive), character, sizeid, id, size handle, handle
func (ph *PacketHandler) sendRankings(server *ServerThread, socket net.Conn, ps *Packet) {
	emptyRankings := []byte{
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x06, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x10, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x00, 0x00, 0x06, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x10, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x00, 0x00, 0x06, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x10, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x00, 0x00, 0x06, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x10, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x00, 0x00, 0x06, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
		0x00, 0x10, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
	}

	// ranking for which area is requested
	emptyRankings[0] = ps.pay[0]
	emptyRankings[1] = ps.pay[1]

	// Create a byte slice of fixed size: 2+8+1+7*4+6*(1+1+2+6+2+16) = 207 bytes.
	buf := make([]byte, 207)
	r := bytes.NewBuffer(buf[:0]) // empty buffer with capacity 207

	// Write values using big-endian order:
	// Write short: scenario from ps.Payload[1]
	if err := binary.Write(r, binary.BigEndian, uint16(ps.pay[1])); err != nil {
		log.Printf("binary.Write error: %v", err)
	}
	// Write int: 111*100
	binary.Write(r, binary.BigEndian, int32(111*100))
	// Write int: ps.Payload[1] as int32
	binary.Write(r, binary.BigEndian, int32(ps.pay[1]))
	// Write byte: 0
	r.WriteByte(0)
	// Write int: 310*10
	binary.Write(r, binary.BigEndian, int32(310*10))
	// Write int: 320*10
	binary.Write(r, binary.BigEndian, int32(320*10))
	// Write int: 330*100 (rank cleartime)
	binary.Write(r, binary.BigEndian, int32(330*100))
	// Write int: 340*100
	binary.Write(r, binary.BigEndian, int32(340*100))
	// Write int: 350
	binary.Write(r, binary.BigEndian, int32(350))
	// Write int: 360*100
	binary.Write(r, binary.BigEndian, int32(360*100))
	// Write int: 370
	binary.Write(r, binary.BigEndian, int32(370))

	// For each of the 6 ranking entries:
	for t := 0; t < 6; t++ {
		r.WriteByte(1)                                // status: 1 = alive
		r.WriteByte(byte(t))                          // character
		binary.Write(r, binary.BigEndian, uint16(6))  // handle length: 6
		r.Write([]byte("HANDLE"))                     // handle (6 bytes)
		binary.Write(r, binary.BigEndian, uint16(16)) // fixed, rest is spaced 0x20
		r.WriteByte(byte(0x41 + t))                   // 1st byte of name to mark
		r.Write([]byte("- RANKTEST     "))            // name (assumed to be 16 bytes)
	}

	// Looks like first half = ranking with resultpoints
	// second half = ranking with cleartimepoints
	emptyRankings = r.Bytes()

	p := NewPacket(commands.RANKINGS, commands.TELL, commands.SERVER, ps.pid, emptyRankings)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAreaCount(server *ServerThread, socket net.Conn, ps *Packet) {
	areacount := []byte{0, 10}

	areacount[0] = byte((ph.areas.GetAreaCount() >> 8) & 0xFF)
	areacount[1] = byte((ph.areas.GetAreaCount()) & 0xFF)

	outp := NewPacket(commands.AREACOUNT, commands.TELL, commands.SERVER, ps.pid, areacount)
	ph.addOutPacket(server, socket, outp)
}

func (ph *PacketHandler) sendAreaPlayerCnt(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0; 0,0; 0xff, 0xff; 0,0
	areaplayercount := []byte{0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0}
	nr := ps.GetNumber()
	cnt := ph.clients.CountPlayersInArea(nr)

	areaplayercount[0] = byte(nr>>8) & 0xff
	areaplayercount[1] = byte(nr) & 0xff
	areaplayercount[2] = byte(cnt[0]>>8) & 0xff
	areaplayercount[3] = byte(cnt[0]) & 0xff
	areaplayercount[4] = byte(cnt[1]>>8) & 0xff
	areaplayercount[5] = byte(cnt[1]) & 0xff
	areaplayercount[8] = byte(cnt[2]>>8) & 0xff
	areaplayercount[9] = byte(cnt[2]) & 0xff

	p := NewPacket(commands.AREAPLAYERCNT, commands.TELL, commands.SERVER, ps.pid, areaplayercount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAreaStatus(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0; 0;
	areastatus := []byte{0, 0, 0}
	nr := ps.GetNumber()

	areastatus[0] = byte(nr>>8) & 0xff
	areastatus[1] = byte(nr) & 0xff
	areastatus[2] = ph.areas.GetStatus(nr)

	p := NewPacket(commands.AREASTATUS, commands.TELL, commands.SERVER, ps.pid, areastatus)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAreaSelect(server *ServerThread, socket net.Conn, ps *Packet) {
	retval := []byte{0, 0}
	nr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, nr, 0, 0)
	retval[0] = byte(nr>>8) & 0xff
	retval[1] = byte(nr) & 0xff

	p := NewPacket(commands.AREASELECT, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)

	ph.broadcastAreaPlayerCnt(server, socket, nr)
}

func (ph *PacketHandler) sendAreaName(server *ServerThread, socket net.Conn, ps *Packet) {
	nr := ps.GetNumber()
	name := ph.areas.GetName(nr)
	namebytes := make([]byte, len(name)+4)
	namebytes[0] = byte(nr>>8) & 0xff
	namebytes[1] = byte(nr) & 0xff
	namebytes[2] = byte(len(name)>>8) & 0xff
	namebytes[3] = byte(len(name)) & 0xff
	copy(namebytes[4:], []byte(name))
	p := NewPacket(commands.AREANAME, commands.TELL, commands.SERVER, ps.pid, namebytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAreaDescript(server *ServerThread, socket net.Conn, ps *Packet) {
	nr := ps.GetNumber()
	desc := ph.areas.GetDescription(nr)
	descbytes := make([]byte, len(desc)+4)
	descbytes[0] = byte(nr>>8) & 0xff
	descbytes[1] = byte(nr) & 0xff
	descbytes[2] = byte(len(desc)>>8) & 0xff
	descbytes[3] = byte(len(desc)) & 0xff
	copy(descbytes[4:], []byte(desc))
	p := NewPacket(commands.AREADESCRIPT, commands.TELL, commands.SERVER, ps.pid, descbytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRoomPlayerCnt(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x01; 0x00,0x00; 0x00,0x03; 0xff,0xff; 0,0
	area := ph.clients.FindClientBySocket(socket).area
	room := ps.GetNumber()
	roomplayercount := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xff, 0xff, 0, 0}
	roomplayercount[0] = byte(room>>8) & 0xff
	roomplayercount[1] = byte(room) & 0xff
	cnt := ph.clients.CountPlayersInRoom(area, room)
	roomplayercount[2] = byte(cnt>>8) & 0xff
	roomplayercount[3] = byte(cnt) & 0xff
	cnt = ph.gameServerPacketHandler.CountInGamePlayers() + ph.clients.CountPlayersInRoom(51, 0)
	roomplayercount[4] = byte(cnt>>8) & 0xff
	roomplayercount[5] = byte(cnt) & 0xff
	p := NewPacket(commands.ROOMPLAYERCNT, commands.TELL, commands.SERVER, ps.pid, roomplayercount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRoomStatus(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x00; 0x00
	retval := []byte{0x00, 0x00, 0x00}
	roomnr := ps.GetNumber()
	area := ph.clients.FindClientBySocket(socket).area
	retval[0] = byte(roomnr>>8) & 0xff
	retval[1] = byte(roomnr) & 0xff
	retval[2] = ph.rooms.GetStatus(area, roomnr)
	p := NewPacket(commands.ROOMSTATUS, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRoomName(server *ServerThread, socket net.Conn, ps *Packet) {
	roomnr := ps.GetNumber()
	area := ph.clients.FindClientBySocket(socket).area
	name := ph.rooms.GetName(area, roomnr)
	namebytes := make([]byte, len(name)+4)
	namebytes[0] = byte(roomnr>>8) & 0xff
	namebytes[1] = byte(roomnr) & 0xff
	namebytes[2] = byte(len(name)>>8) & 0xff
	namebytes[3] = byte(len(name)) & 0xff
	copy(namebytes[4:], []byte(name))
	p := NewPacket(commands.ROOMNAME, commands.TELL, commands.SERVER, ps.pid, namebytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) send6308(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x01; 0x00,0x02; 0x81,0x40
	retval := []byte{0x00, 0x01, 0x00, 0x02, 0x81, 0x40}
	retval[0] = ps.pay[0]
	retval[1] = ps.pay[1]
	p := NewPacket(commands.UNKN6308, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRoomsCount(server *ServerThread, socket net.Conn, ps *Packet) {
	countbytes := []byte{0, 0}
	count := ph.rooms.GetRoomCount()

	countbytes[0] = byte(count>>8) & 0xff
	countbytes[1] = byte(count) & 0xff

	p := NewPacket(commands.ROOMSCOUNT, commands.TELL, commands.SERVER, ps.pid, countbytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendEnterRoom(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x00
	retval := []byte{0, 0}
	roomnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	cl.room = roomnr
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, area, roomnr, 0)
	retval[0] = byte(roomnr>>8) & 0xff
	retval[1] = byte(roomnr) & 0xff
	p := NewPacket(commands.ENTERROOM, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
	ph.broadcastRoomPlayerCnt(server, area, roomnr)
}

func (ph *PacketHandler) broadcastRoomPlayerCnt(server *ServerThread, area, room int) {
	// 0x00,0x01; 0x00,0x00; 0x00,0x03; 0xff,0xff; 0,0
	roomplayercount := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xff, 0xff, 0, 0}
	roomplayercount[0] = byte(room>>8) & 0xff
	roomplayercount[1] = byte(room) & 0xff
	cnt := ph.clients.CountPlayersInRoom(area, room)
	roomplayercount[2] = byte(cnt>>8) & 0xff
	roomplayercount[3] = byte(cnt) & 0xff
	cnt = ph.gameServerPacketHandler.CountInGamePlayers() + ph.clients.CountPlayersInRoom(51, 0)
	roomplayercount[4] = byte(cnt>>8) & 0xff
	roomplayercount[5] = byte(cnt) & 0xff
	p := NewPacket(commands.ROOMPLAYERCNT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), roomplayercount)
	ph.broadcastInArea(server, p, area)
}

func (ph *PacketHandler) broadcastInArea(server *ServerThread, p *Packet, area int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.area == area && cl.room == 0 {
			ph.addOutPacket(server, cl.socket, p)
		}
	}
}

func (ph *PacketHandler) sendSlotCount(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0
	slotcount := []byte{0, 0}
	cnt := ph.slots.GetSlotCount()
	slotcount[0] = byte(cnt>>8) & 0xff
	slotcount[1] = byte(cnt) & 0xff
	p := NewPacket(commands.SLOTCOUNT, commands.TELL, commands.SERVER, ps.pid, slotcount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendSlotPlayerStatus(server *ServerThread, socket net.Conn, ps *Packet) {
	//0x00,0x00; 0x00,0x00; 0x00,0x00; 0x00,0x00, 0x00,0x00
	retval := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := ps.GetNumber()
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[3] = byte(ph.clients.CountPlayersInSlot(area, room, slotnr))
	retval[5] = byte(0) // TODO: what is this value?
	retval[7] = byte(ph.slots.GetMaximumPlayers(area, room, slotnr))
	retval[9] = retval[3] // TODO: what is playin2?
	p := NewPacket(commands.SLOTPLRSTATUS, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendSlotTitle(server *ServerThread, socket net.Conn, ps *Packet) {
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room

	var slotname []byte
	// character test slot. maybe remove? not sure TODO
	if area == 0x002 && room == 0x001 && slotnr == 0x003 {
		slotname = []byte("Testgame")
	} else {
		slotname = ph.slots.GetName(area, room, slotnr)
	}

	slotnamebytes := make([]byte, len(slotname)+4)
	slotnamebytes[0] = byte(slotnr>>8) & 0xff
	slotnamebytes[1] = byte(slotnr) & 0xff
	slotnamebytes[2] = byte(len(slotname)>>8) & 0xff
	slotnamebytes[3] = byte(len(slotname)) & 0xff
	copy(slotnamebytes[4:], []byte(slotname))
	p := NewPacket(commands.SLOTTITLE, commands.TELL, commands.SERVER, ps.pid, slotnamebytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendSlotAttrib2(server *ServerThread, socket net.Conn, ps *Packet) {
	retval := []byte{
		0, 1, // slot nr
		0, 4, // max players for slot
		0, 4,
		0, 1,
		0, 4,
		0, 1,
	}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[3] = ph.slots.GetMaximumPlayers(area, room, slotnr)
	// TODO: what do these attributes mean? extend slots get/set with those
	p := NewPacket(commands.SLOTATTRIB2, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPasswdProtect(server *ServerThread, socket net.Conn, ps *Packet) {
	//0,1; 0
	retval := []byte{0, 1, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[2] = ph.slots.GetProtection(area, room, slotnr)
	p := NewPacket(commands.SLOTPWDPROT, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendSlotSceneType(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0; 0,0; 0,0
	retval := []byte{0, 0, 0, 0, 0, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[3] = ph.slots.GetSlotType(area, room, slotnr)
	retval[5] = ph.slots.GetScenario(area, room, slotnr)
	p := NewPacket(commands.SLOTSCENTYPE, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendCreateSlot(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := ps.GetNumber()

	answer := []byte{0, 0}
	cl.slot = slotnr
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, area, room, slotnr)
	cl.host = 1
	cl.player = 1
	ph.slots.GetSlot(area, room, slotnr).SetStatus(STATUS_INCREATE)
	ph.slots.GetSlot(area, room, slotnr).SetLivetime()

	ph.slots.GetSlot(area, room, slotnr).SetHost(cl.userID)

	ph.broadcastSlotPlayerStatus(server, area, room, slotnr)
	ph.broadcastSlotStatus(server, area, room, slotnr)

	answer[1] = byte(slotnr) & 0xff
	p := NewPacket(commands.CREATESLOT, commands.TELL, commands.SERVER, ps.pid, answer)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRulesCount(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0
	rulescount := []byte{0}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := ps.GetNumber()
	rulescount[0] = byte(ph.slots.GetRulesCount(area, room, slotnr))
	p := NewPacket(commands.RULESCOUNT, commands.TELL, commands.SERVER, ps.pid, rulescount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRuleAttCount(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0

	ruleattcount := []byte{0, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	ruleattcount[0] = ps.pay[2]
	ruleattcount[1] = ph.slots.GetRulesAttCount(area, room, slotnr, int(ps.pay[2]))
	p := NewPacket(commands.RULEATTCOUNT, commands.TELL, commands.SERVER, ps.pid, ruleattcount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) send6602(server *ServerThread, socket net.Conn, ps *Packet) {
	//1; 0,0
	retval := []byte{1, 0, 0}
	nr := ps.GetNumber()
	retval[1] = byte(nr>>8) & 0xff
	retval[2] = byte(nr) & 0xff
	p := NewPacket(commands.UNKN6602, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) send6601(server *ServerThread, socket net.Conn, ps *Packet) {
	//1; 0,0
	retval := []byte{1, 0, 0}
	nr := ps.GetNumber()
	retval[1] = byte(nr>>8) & 0xff
	retval[2] = byte(nr) & 0xff
	p := NewPacket(commands.UNKN6601, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRuleDescript(server *ServerThread, socket net.Conn, ps *Packet) {
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	rule := ps.pay[2]
	rulename := ph.slots.GetRuleName(area, room, slotnr, int(rule))
	rulenamebytes := make([]byte, len(rulename)+3)
	rulenamebytes[0] = rule
	rulenamebytes[1] = byte(len(rulename)>>8) & 0xff
	rulenamebytes[2] = byte(len(rulename)) & 0xff
	copy(rulenamebytes[3:], []byte(rulename))
	p := NewPacket(commands.RULEDESCRIPT, commands.TELL, commands.SERVER, ps.pid, rulenamebytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRuleValue(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0
	retval := []byte{0, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	rule := ps.pay[2]
	retval[0] = rule
	retval[1] = ph.slots.GetRuleValue(area, room, slotnr, int(rule))
	p := NewPacket(commands.RULEVALUE, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendRuleAttrib(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0
	retval := []byte{0, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	rule := ps.pay[2]
	ruleatt := ph.slots.GetRuleAttribute(area, room, slotnr, int(rule))
	retval[0] = rule
	retval[1] = ruleatt
	p := NewPacket(commands.RULEATTRIB, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

/* 3/17 - this function is currently broken, or maybe
it's one of the functions it's calling, or maybe it's
responding to a bad packet or something. who knows...
it seems like it's calling GetStatus with slotnr = 0
which leads to inputting a negative value and thus trying
to access a negative index in the status array.

3/18 - i "fixed" this by deviating from how the java code
initializes its values. essentially, i "zero indexed" areas
and slot numbers. i haven't seen any drawbacks to this yet,
but i have a suspicion that there's a reason it was 1-indexed
in the original java code...*/

func (ph *PacketHandler) sendSlotStatus(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x00; 0x00
	slotnr := ps.GetNumber()
	fmt.Printf("PacketHandler sendSlotStatus() slotnr: %d\n", slotnr) // Debug print
	slotstatus := []byte{0x00, 0x00, 0x00}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	fmt.Printf("PacketHandler sendSlotStatus() area: %d, room: %d\n", area, room) // Debug print
	slotstatus[0] = byte(slotnr>>8) & 0xff
	slotstatus[1] = byte(slotnr) & 0xff
	slotstatus[2] = ph.slots.GetStatus(area, room, slotnr)
	p := NewPacket(commands.SLOTSTATUS, commands.TELL, commands.SERVER, ps.pid, slotstatus)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) broadcastAreaPlayerCnt(server *ServerThread, socket net.Conn, nr int) {
	// 0,0; 0,0; 0xff,0xff; 0,0
	areaplayercount := []byte{0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0}
	cnt := ph.clients.CountPlayersInArea(nr)
	cnt[2] = cnt[2] + ph.clients.CountPlayersInRoom(51, 0) + ph.gameServerPacketHandler.CountInGamePlayers()

	areaplayercount[0] = byte(nr>>8) & 0xff
	areaplayercount[1] = byte(nr) & 0xff
	areaplayercount[2] = byte(cnt[0]>>8) & 0xff
	areaplayercount[3] = byte(cnt[0]) & 0xff
	areaplayercount[4] = byte(cnt[1]>>8) & 0xff
	areaplayercount[5] = byte(cnt[1]) & 0xff
	areaplayercount[8] = byte(cnt[2]>>8) & 0xff
	areaplayercount[9] = byte(cnt[2]) & 0xff

	p := NewPacket(commands.AREAPLAYERCNT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), areaplayercount)
	ph.broadcastInAreaNAreaSelect(server, p, nr)
}

func (ph *PacketHandler) broadcastInAreaNAreaSelect(server *ServerThread, p *Packet, area int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.area == area || (cl.area == 0 && cl.room == 0) {
			// TODO: original java source touches the queue directly here
			// should we do the same or use addOutPacket?
			ph.addOutPacket(server, cl.socket, p)
		}
	}
}

func (ph *PacketHandler) broadcastSlotPlayerStatus(server *ServerThread, area, room, slot int) {
	// 0x00,0x00; 0x00,0x00; 0x00,0x00; 0x00,0x00, 0x00,0x00
	retval := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	retval[0] = byte(slot>>8) & 0xff
	retval[1] = byte(slot) & 0xff
	retval[3] = byte(ph.clients.CountPlayersInSlot(area, room, slot))
	retval[5] = 0 // TODO: what is this value?
	retval[7] = byte(ph.slots.GetMaximumPlayers(area, room, slot))
	retval[9] = retval[3] // TODO: what is playin2?
	p := NewPacket(commands.SLOTPLRSTATUS, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), retval)
	ph.broadcastInSlotNRoom(server, p, area, room, slot)
}

func (ph *PacketHandler) broadcastInSlotNRoom(server *ServerThread, p *Packet, area, room, slot int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.area == area && cl.room == room && (cl.slot == slot || cl.slot == 0) {
			// TODO: original java source touches the queue directly here
			// should we do the same or use addOutPacket?
			ph.addOutPacket(server, cl.socket, p)
		}
	}
}

func (ph *PacketHandler) broadcastSlotStatus(server *ServerThread, area, room, slot int) {
	// 0x00,0x00; 0x00
	retval := []byte{0x00, 0x00, 0x00}
	retval[0] = byte(slot>>8) & 0xff
	retval[1] = byte(slot) & 0xff
	retval[2] = ph.slots.GetStatus(area, room, slot)
	p := NewPacket(commands.SLOTSTATUS, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), retval)
	ph.broadcastInSlotNRoom(server, p, area, room, slot)
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

func (ph *PacketHandler) send6002(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)

	// area := cl.area
	// room := cl.room
	// slot := cl.slot

	// reset client's area/slot

	cl.area = 0
	cl.room = 0
	cl.slot = 0
	cl.player = 0

	//free slot for other players when last player left
	// need to implement theese:
	// if(clients.countPlayersInSlot(area, room, slot) == 0) {
	// 	slots.getSlot(area, room, slot).reset();
	// 	this.broadcastSlotPlayerStatus(server, area, room, slot);
	// 	this.broadcastPasswdProtect(server, area, room, slot);
	// 	this.broadcastSlotTitle(server, area, room, slot);
	// 	this.broadcastSlotSceneType(server, area, room, slot);
	// 	this.broadcastSlotAttrib2(server, area, room, slot);
	// 	this.broadcastSlotStatus(server, area, room, slot);
	// }

	p := NewPacketWithoutPayload(commands.UNKN6002, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) RemoveClientNoDisconnect(server *ServerThread, socket net.Conn) {
	cl := ph.clients.FindClientBySocket(socket)

	if cl == nil {
		return
	}
	cl.ConnAlive = false

	fmt.Printf("PacketHandler RemoveClientNoDisconnect() client: %s socket: %p\n", cl.userID, socket)
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
