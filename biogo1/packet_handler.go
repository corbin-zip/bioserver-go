package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"main/commands"
	"net"
	"os"

	// "time"
	"path/filepath"
	"runtime"
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
	logger                  *log.Logger
	information             *Information
	gsIP                    []byte
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
	ph.logger = log.New(os.Stdout, "", log.Ltime)
	ph.information = NewInformation()
	ph.gsIP = []byte{192, 168, 1, 135}
	return ph
}

func (ph *PacketHandler) debug(format string, v ...interface{}) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
	} else {
		funcName = "???"
	}
	file = filepath.Base(file)
	msg := fmt.Sprintf(format, v...)

	ph.logger.Printf("%s:%d %s() %s", file, line, funcName, msg)
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
func (ph *PacketHandler) getNextPacketID() int {
	ph.packetIDCounter++
	return ph.packetIDCounter
}

func (ph *PacketHandler) getNextGameNumber() int {
	ph.gameNumber++
	return ph.gameNumber
}

func (ph *PacketHandler) addOutPacket(server *ServerThread, socket net.Conn, p *Packet) {
	// ph.debug("PacketHandler addOutPacket() 0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))
	ph.debug("0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))

	event := ServerDataEvent{
		server: server,
		socket: socket,
		data:   p.GetPacketData(),
	}
	select {
	case ph.queue <- event:
	default:
		ph.debug("PacketHandler queue full\n")
	}

}

func (ph *PacketHandler) ProcessData(server *ServerThread, socket net.Conn, data []byte) {
	offset := 0
	remaining := len(data)

	for remaining > 0 {
		if remaining < HEADER_SIZE {
			ph.debug("Incomplete packet: %d bytes remaining\n", remaining)
			break // or handle error
		}

		// Peek into header to get payload length
		// Assumes payload length is stored in data[4] and data[5]
		pLen := int(data[offset+4])<<8 | int(data[offset+5])
		packetSize := HEADER_SIZE + pLen
		if remaining < packetSize {
			ph.debug("Incomplete packet: %d bytes remaining\n", remaining)
			break // incomplete packet; wait for more data
		}
		packetData := data[offset : offset+packetSize]
		p := NewPacketFromBytes(packetData)

		// ph.debug("PacketHandler ProcessData() - In cmd: 0x%X (%s)\n", p.cmd, commands.GetConstName(p.cmd))
		// if p.cmd == commands.LOGIN && ph.clients.FindClientBySocket(socket) != nil {
		// fmt.Println("PacketHandler ProcessData() Dropping duplicate login packet")
		// } else {
		ph.HandleInPacket(server, socket, p)
		// }

		offset += packetSize
		remaining -= packetSize

		// ph.debug("PacketHandler ProcessData() %p: Processed packet of size %d, remaining: %d\n", socket, packetSize, remaining)
	}

}

func (ph *PacketHandler) HandleInPacket(server *ServerThread, socket net.Conn, packet *Packet) {
	p := packet
	// ph.debug("PacketHandler HandleInPacket() 0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))
	ph.debug("0x%X - who: %s cmd: %s qsw: %s\n", p.cmd, commands.GetConstName(p.who), commands.GetConstName(p.cmd), commands.GetConstName(p.qsw))

	switch packet.who {
	case commands.CLIENT:
		switch packet.qsw {
		case commands.QUERY:
			switch packet.cmd {
			case commands.UNKN61A0:
				ph.sendTimeout(server, socket, packet)
			case commands.CHECKRND:
				ph.sendCheckRnd(server, socket, packet)
			case commands.UNKN61A1:
				ph.send61A1(server, socket, packet)
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
			case commands.ATTRDESCRIPT:
				ph.sendAttrDescript(server, socket, packet)
			case commands.ATTRATTRIB:
				ph.sendAttrAttrib(server, socket, packet)
			case commands.PLAYERSTATS:
				ph.sendPlayerStats(server, socket, packet)
			case commands.EXITSLOTLIST:
				ph.sendExitSlotlist(server, socket, packet)
			case commands.EXITAREA:
				ph.sendExitArea(server, socket, packet)
			case commands.CREATESLOT:
				ph.sendCreateSlot(server, socket, packet)
			case commands.SCENESELECT:
				ph.sendSceneSelect(server, socket, packet)
			case commands.SLOTNAME:
				ph.sendSlotName(server, socket, packet)
			case commands.SETRULE:
				ph.sendSetRule(server, socket, packet)
			case commands.UNKN660C:
				ph.send660C(server, socket, packet)
			case commands.SLOTTIMER:
				ph.sendSlotTimer(server, socket, packet)
			case 0x6412:
				ph.send6412(server, socket, packet)
			case 0x6504:
				ph.send6504(server, socket, packet)
			case commands.CANCELSLOT:
				ph.sendCancelSlot(server, socket, packet)
			case commands.SLOTPASSWD:
				ph.sendSlotPasswd(server, socket, packet)
			case commands.PLAYERCOUNT:
				ph.sendPlayerCount(server, socket, packet)
			case commands.PLAYERNUMBER:
				ph.sendPlayerNumber(server, socket, packet)
			case commands.PLAYERSTAT:
				ph.sendPlayerStat(server, socket, packet)
			case commands.PLAYERSCORE:
				ph.sendPlayerScore(server, socket, packet)
			case commands.GAMESESSION:
				ph.sendGameSession(server, socket, packet)
			case commands.GAMEDIFF:
				ph.sendDifficulty(server, socket, packet)
			case commands.GSINFO:
				ph.sendGSinfo(server, socket, packet)
			case commands.ENTERAGL:
				ph.sendEnterAGL(server, socket, packet)
			case commands.AGLSTATS:
				ph.sendAGLstats(server, socket, packet)
			case commands.AGLPLAYERCNT:
				ph.sendAGLplayerCnt(server, socket, packet)
			case commands.LEAVEAGL:
				ph.sendLeaveAGL(server, socket, packet)
			case commands.JOINGAME:
				ph.sendJoinGame(server, socket, packet)
			case commands.GETINFO:
				ph.sendGetInfo(server, socket, packet)
			case commands.EVENTDAT:
				ph.sendEventDat(server, socket, packet)
			case commands.BUDDYLIST:
				ph.sendBuddyList(server, socket, packet)
			case commands.CHECKBUDDY:
				ph.sendCheckBuddy(server, socket, packet)
			case commands.PRIVATEMSG:
				ph.sendPrivateMsg(server, socket, packet)
			case commands.UNKN6181:
				ph.send6181(server, socket, packet)
			case commands.LOGOUT:
				ph.sendLogout(server, socket, packet)
			default:
				ph.debug("Unknown or unimplemented command on query: 0x%X (%s)\n", packet.cmd, commands.GetConstName(packet.cmd))
			}
		case commands.TELL:
			switch packet.cmd {
			case commands.CONNCHECK:
				cl := ph.clients.FindClientBySocket(socket)
				if cl != nil {
					cl.ConnAlive = true
				}
			case commands.LOGIN:
				if ph.checkSession(server, socket, packet) {
					ph.debug("Session check passed!\n")
					// correct session established
					// next step is the version check for File#1 updates
					ph.sendVersionCheck(server, socket)
				} else {
					ph.debug("Session check failed!")
				}
			case commands.CHECKVERSION:
				if ph.checkPatchLevel(server, socket, packet) {
					// if version is older than actual patch, send patch
					ph.debug("literally never reaches here....")
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
				ph.debug("Unknown command on answer: %d (0x%X)\n", packet.cmd, packet.cmd)
			}
		case commands.BROADCAST:
			switch packet.cmd {
			case commands.STARTGAME:
				ph.broadcastGetReady(server, socket)
			case commands.CHATIN:
			    ph.broadcastChatOut(server, socket, packet)
			default:
				ph.debug("Unknown command on broadcast: %d (0x%X)\n", packet.cmd, packet.cmd)
			}
		default:
			ph.debug("Unknown qsw type on incoming packet! 0x%X\n", packet.qsw)
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
		ph.debug("PacketHandler checkSession() Error getting user id:%v\n", err)
		return false
	}

	ph.debug("Session: %s with UserID: %s\n", session, userid)

	if userid != "" {
		// loop through clients and remove old connections
		// then setup client object for this user/session
		// TODO: (should this be implemented by clients.go instead?)
		cl := ph.clients.FindClientByUserID(userid)
		if cl != nil {
			ph.debug("Found a client with UserID %s: %v\n", userid, cl)
			for _, c := range ph.clients.GetList() {
				if c.userID == userid {
					ph.debug("removing client with UserID %s & socket %p\n", userid, c.socket)
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
	ph.debug("Packet data: %+v\n", packetData)
	version := p.GetVersion()
	ph.debug("Decrypted client version: %s\n", version)

	// check if the client has the latest patch level
	// if not, send patch
	return false
}

func (ph *PacketHandler) sendIDHNPairs(server *ServerThread, socket net.Conn) {
	// get the handles tied to this userid (max 3)
	userid := ph.clients.FindClientBySocket(socket).userID

	hn := ph.db.GetHNPairs(userid)

	ph.debug("Sending HNPairs: %v\n", hn)

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
		ph.debug("Error getting game number for userid %s: %v\n", userid, err)
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
	message, err := ph.db.GetMOTD()
	if err != nil {
		message = "error getting motd..."
		ph.debug("Failed to get MOTD: %v\n", err)
	}
	// TODO note to self: ip.addr == 192.168.1.135 and not ssh and data.data contains 61:4C
	// should be 1 byte number (1 apparently), 2 byte length (only of motd apparently), then motd
	message = fmt.Sprintf("<LF=6><BODY><CENTER>%s<END>", message)
	ph.debug("sending MOTD message: %s\n", message)
	motd := NewMOTD(1, message)
	motdp := NewPacket(commands.MOTHEDAY, commands.TELL, commands.SERVER, p.pid, motd.GetPacket())
	ph.addOutPacket(server, socket, motdp)
}

func (ph *PacketHandler) sendCharSelect(server *ServerThread, socket net.Conn, p *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	cl.SetCharacterStats(p.GetCharacterStats())

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
	cl.area = nr
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
	ph.debug("\n\n\n\n\nRequested area name for area %d: %s\n\n\n\n\n", nr, name)
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
	ph.debug("\nentering area %d room %d\n\n", area, roomnr)
	retval[0] = byte(roomnr>>8) & 0xff
	retval[1] = byte(roomnr) & 0xff
	p := NewPacket(commands.ENTERROOM, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
	ph.broadcastRoomPlayerCnt(server, area, roomnr)
}

// this is closer to how the java code does the bytebuffer stuff
// TODO: maybe look at replacing other areas of the go code with this strategy
func (ph *PacketHandler) broadcastChatOut(server *ServerThread, socket net.Conn, ps *Packet) {
    cl := ph.clients.FindClientBySocket(socket)
    area := cl.area
    room := cl.room
    slot := cl.slot

    var broadcast bytes.Buffer

    // who is sending the message
	broadcast.Write(cl.hnPair.GetHNPair())

    // copy message and save to database
    mess := ps.GetChatOutData()

    binary.Write(&broadcast, binary.BigEndian, int16(len(mess)))
    broadcast.Write(mess)
    broadcast.WriteByte(0)
    binary.Write(&broadcast, binary.BigEndian, int32(0x000000ff))

    r := broadcast.Bytes()

	p := NewPacket(commands.CHATOUT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), r)

    if slot > 0 {
        ph.broadcastInSlot(server, p, area, room, slot)
    } else if area != 0 && area != 51 {
        ph.broadcastInArea(server, p, area)
    } else if cl.GameNumber > 0 {
        ph.broadcastInAgl(server, p, cl.GameNumber)
    }
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

func (ph *PacketHandler) broadcastSlotAttrib2(server *ServerThread, area, room, slotnr int) {
	retval := []byte{
		0, 1, // slot nr
		0, 4, // max players for slot
		0, 4,
		0, 1,
		0, 4,
		0, 1,
	}
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[3] = ph.slots.GetMaximumPlayers(area, room, slotnr)
	// TODO: what do these attributes mean? extend slots get/set with those
	p := NewPacket(commands.SLOTATTRIB2, commands.TELL, commands.SERVER, ph.getNextPacketID(), retval)
	ph.broadcastInSlotNRoom(server, p, area, room, slotnr)
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

func (ph *PacketHandler) broadcastSlotSceneType(server *ServerThread, area, room, slotnr int) {
	// 0,0; 0,0; 0,0
	retval := []byte{0, 0, 0, 0, 0, 0}
	retval[0] = byte(slotnr>>8) & 0xff
	retval[1] = byte(slotnr) & 0xff
	retval[3] = ph.slots.GetSlotType(area, room, slotnr)
	retval[5] = ph.slots.GetScenario(area, room, slotnr)
	p := NewPacket(commands.SLOTSCENTYPE, commands.TELL, commands.SERVER, ph.getNextPacketID(), retval)
	ph.broadcastInSlotNRoom(server, p, area, room, slotnr)
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

func (ph *PacketHandler) sendAttrAttrib(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0, 0
	retval := []byte{0, 0, 0}
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	rule := ps.pay[2]
	attnr := ps.pay[3]
	attr := ph.slots.GetRuleAttributeAtt(area, room, slotnr, int(rule), int(attnr))
	retval[0] = rule
	retval[1] = attnr
	retval[2] = attr
	p := NewPacket(commands.ATTRATTRIB, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAttrDescript(server *ServerThread, socket net.Conn, ps *Packet) {
	slotnr := ps.GetNumber()
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	rulenr := ps.pay[2]
	attnr := ps.pay[3]
	attdesc := ph.slots.GetRuleAttributeDescription(area, room, slotnr, int(rulenr), int(attnr))
	retbytes := make([]byte, len(attdesc)+4)
	retbytes[0] = rulenr
	retbytes[1] = attnr
	retbytes[2] = byte(len(attdesc)>>8) & 0xff
	retbytes[3] = byte(len(attdesc)) & 0xff
	copy(retbytes[4:], []byte(attdesc))
	p := NewPacket(commands.ATTRDESCRIPT, commands.TELL, commands.SERVER, ps.pid, retbytes)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPlayerStats(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := ps.GetNumber()
	playerstats := ph.clients.GetPlayerStats(area, room, slotnr)

	// Special character test slot handling TODO
	if area == 0x002 && room == 0x001 && slotnr == 0x003 {
		c := playerstats[3]
		ptr := 4
		for t := 0; t < int(c); t++ {
			off := int(playerstats[ptr+1])
			ptr = ptr + 2 + off // handle
			off = int(playerstats[ptr+1])
			ptr = ptr + 2 + off // nickname
			off = int(playerstats[ptr+1] & 0xff)
			ptr = ptr + 2 + off       // statistics
			playerstats[ptr-8] = 0xff // 0x6a; // dummy value
		}
	}

	p := NewPacket(commands.PLAYERSTATS, commands.TELL, commands.SERVER, ps.pid, playerstats)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendExitSlotlist(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	cl.room = 0
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, area, room, 0)

	p := NewPacketWithoutPayload(commands.EXITSLOTLIST, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)

	ph.broadcastRoomPlayerCnt(server, area, room)
}

func (ph *PacketHandler) sendExitArea(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	cl.area = 0

	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, 0, 0, 0)

	p := NewPacketWithoutPayload(commands.EXITAREA, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)

	ph.broadcastAreaPlayerCnt(server, socket, area)
}

func (ph *PacketHandler) sendSlotName(server *ServerThread, socket net.Conn, ps *Packet) {
	// TODO: this feels buggy
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	slottitle := ps.GetDecryptedString()
	ph.debug("\nSetting slot title for area %d room %d slot %d to %s\n\n", area, room, slotnr, slottitle)
	ph.slots.GetSlot(area, room, slotnr).SetName(slottitle)
	p := NewPacket(commands.SLOTNAME, commands.TELL, commands.SERVER, ps.pid, ps.pay)
	ph.addOutPacket(server, socket, p)
	ph.broadcastSlotTitle(server, area, room, slotnr)

}

func (ph *PacketHandler) broadcastSlotTitle(server *ServerThread, area, room, slot int) {
	// 0x00,0x00; 0x00,0x00; 0x00,0x00; 0x00,0x00, 0x00,0x00
	slottitle := ph.slots.GetSlot(area, room, slot).name
	broadcast := make([]byte, len(slottitle)+4)
	broadcast[0] = byte(slot>>8) & 0xff
	broadcast[1] = byte(slot) & 0xff
	broadcast[2] = byte(len(slottitle)>>8) & 0xff
	broadcast[3] = byte(len(slottitle)) & 0xff
	copy(broadcast[4:], []byte(slottitle))
	p := NewPacket(commands.SLOTTITLE, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), broadcast)
	ph.broadcastInSlotNRoom(server, p, area, room, slot)
}

func (ph *PacketHandler) sendSetRule(server *ServerThread, socket net.Conn, ps *Packet) {
	retval := []byte{0}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	rulenr := ps.pay[0]
	ruleval := ps.pay[1]

	slot := ph.slots.GetSlot(area, room, slotnr)
	slot.SetRuleValue(int(rulenr), ruleval)

	p := NewPacket(commands.SETRULE, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) send660C(server *ServerThread, socket net.Conn, ps *Packet) {
	p := NewPacket(commands.UNKN660C, commands.TELL, commands.SERVER, ps.pid, ps.pay)
	ph.addOutPacket(server, socket, p)
}

// 1st word is slottype: 0011 = dvd, 0012 = hdd
// 2nd word are the scenes: wild things 0001, underbelly 0002, flashback 0003, desperate times 0004
func (ph *PacketHandler) sendSceneSelect(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,0; 0,0x12; 0,2
	scenetype := []byte{0, 0, 0, 0x12, 0, 2}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot

	scenetype[1] = byte(slotnr) & 0xff // slot
	scenetype[3] = ps.pay[1]           // type
	scenetype[5] = ps.pay[3]           // scenario

	slot := ph.slots.GetSlot(area, room, slotnr)
	slot.SetSlotType(scenetype[3])
	slot.SetScenario(scenetype[5])

	p := NewPacket(commands.SCENESELECT, commands.TELL, commands.SERVER, ps.pid, scenetype)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendSlotTimer(server *ServerThread, socket net.Conn, ps *Packet) {
	timing := []byte{0, 0, 7, 8}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	livetime := ph.slots.GetSlot(area, room, slotnr).GetLivetime()

	timing[0] = byte(slotnr) & 0xff
	timing[2] = byte(livetime>>8) & 0xff
	timing[3] = byte(livetime) & 0xff

	p := NewPacket(commands.SLOTTIMER, commands.TELL, commands.SERVER, ps.pid, timing)
	ph.addOutPacket(server, socket, p)

	if livetime == 0 {
		ph.broadcastGetReady(server, socket)
	}
}

func (ph *PacketHandler) broadcastGetReady(server *ServerThread, socket net.Conn) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	gamenr := ph.slots.GetSlot(area, room, slotnr).gamenr

	// if this slot has no gamenumber, create one
	if gamenr == 0 {
		// create a gamesession and save it to the clients in slot
		// used for gameserver and after game lobby
		gamenr = ph.getNextGameNumber()
		for _, c := range ph.clients.GetList() {
			// TODO double check this
			if c.area == area && c.room == room && c.slot == slotnr {
				c.GameNumber = gamenr
				ph.db.UpdateClientGame(c.userID, gamenr)
			}
		}
		ph.slots.GetSlot(area, room, slotnr).gamenr = gamenr
	}

	ph.slots.GetSlot(area, room, slotnr).SetStatus(STATUS_BUSY)
	ph.broadcastSlotStatus(server, area, room, slotnr)

	p := NewPacketWithoutPayload(commands.GETREADY, commands.BROADCAST, commands.SERVER, ph.getNextPacketID())
	ph.broadcastInSlot(server, p, area, room, slotnr)
}

func (ph *PacketHandler) broadcastInSlot(server *ServerThread, p *Packet, area, room, slot int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.area == area && cl.room == room && cl.slot == slot {
			ph.addOutPacket(server, cl.socket, p)
		}
	}
}

func (ph *PacketHandler) send6412(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0,1; 0,0,0,0
	answer := []byte{0, 1, 0, 0, 0, 0}
	nr := ps.GetNumber()
	answer[1] = byte(nr) & 0xff
	p := NewPacket(commands.UNKN6412, commands.TELL, commands.SERVER, ps.pid, answer)
	ph.addOutPacket(server, socket, p)
}

// last packet from slot creator !!
func (ph *PacketHandler) send6504(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot

	retval := []byte{1}

	// just in case
	retval[0] = ps.pay[0]

	// set usage and playerstatus

	if cl.host == 1 {
		ph.slots.GetSlot(area, room, slotnr).SetStatus(STATUS_GAMESET)
		ph.slots.GetSlot(area, room, slotnr).SetLivetime()
	}

	ph.broadcastSlotPlayerStatus(server, area, room, slotnr)
	ph.broadcastPasswdProtect(server, area, room, slotnr)
	ph.broadcastSlotSceneType(server, area, room, slotnr)
	ph.broadcastSlotAttrib2(server, area, room, slotnr)
	ph.broadcastSlotStatus(server, area, room, slotnr)
	ph.broadcastPlayerOK(server, socket)

	p := NewPacket(commands.UNKN6504, commands.TELL, commands.SERVER, ps.pid, retval)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendJoinGame(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room

	retslot := []byte{0, 1}
	slotnr := ps.GetNumber()
	ph.debug("\n\n!!!!!!!!!!!!!!!!\n\nJoining game in area %d room %d slot %d\n\n!!!!!!!!!!!!\n\n", area, room, slotnr)
	ph.debug("our packet pay[0] and pay[1] were %d and %d\n\n\n\n", ps.pay[0], ps.pay[1])

	// check if slot is free
	if ph.slots.GetStatus(area, room, slotnr) == STATUS_BUSY {
		mess := NewPacketString("<LF=6><BODY><CENTER>game is full<END>").GetData()
		p := NewPacket(commands.JOINGAME, commands.TELL, commands.SERVER, ps.pid, mess)
		p.SetErr()
		ph.addOutPacket(server, socket, p)
		return
	}

	slot := ph.slots.GetSlot(area, room, slotnr)

	// check if we can join the slot
	if slot.GetStatus() != STATUS_GAMESET {
		mess := NewPacketString("<LF=6><BODY><CENTER>not possible<END>").GetData()
		p := NewPacket(commands.JOINGAME, commands.TELL, commands.SERVER, ps.pid, mess)
		p.SetErr()
		ph.addOutPacket(server, socket, p)
		return
	}

	// hostuser := ph.slots.GetSlot(area, room, slotnr).GetHost()
	// clntuser := cl.userID

	// get password, check it
	pass := ps.GetPassword()
	if bytes.Equal(pass, slot.GetPassword()) || slot.GetProtection() == PROTECTION_OFF {
		retslot[1] = byte(slotnr) & 0xff

		// assign a player number, set slot
		player := ph.clients.GetFreePlayerNum(area, room, slotnr)
		ph.debug("!!!!!!!!!!!!!!!!!!!!!!!!\n\n\nfree player number is %d\n", player)
		cl.slot = slotnr
		cl.player = byte(player)
		ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, area, room, slotnr)

		p := NewPacket(commands.JOINGAME, commands.TELL, commands.SERVER, ps.pid, retslot)
		ph.addOutPacket(server, socket, p)

		n := ph.clients.CountPlayersInSlot(area, room, slotnr)
		if n >= int(ph.slots.GetMaximumPlayers(area, room, slotnr)) {
			ph.slots.GetSlot(area, room, slotnr).SetStatus(STATUS_BUSY)
		}

		ph.broadcastSlotPlayerStatus(server, area, room, slotnr)
		ph.broadcastSlotStatus(server, area, room, slotnr)
		ph.broadcastSlotAttrib2(server, area, room, slotnr)

		// broadcast stats of new player
		status := cl.GetCharacterStat()

		p2 := NewPacket(commands.PLAYERSTATBC, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), status)

		ph.broadcastInSlot(server, p2, area, room, slotnr)
	} else {
		mess := NewPacketString("<LF=6><BODY><CENTER>wrong password<END>").GetData()
		p := NewPacket(commands.JOINGAME, commands.TELL, commands.SERVER, ps.pid, mess)
		p.SetErr()
		ph.addOutPacket(server, socket, p)
	}

}
func (ph *PacketHandler) broadcastPasswdProtect(server *ServerThread, area, room, slot int) {
	// 0,1; 0
	retval := []byte{0, 1, 0}
	retval[0] = byte(slot>>8) & 0xff
	retval[1] = byte(slot) & 0xff
	retval[2] = ph.slots.GetProtection(area, room, slot)
	p := NewPacket(commands.SLOTPWDPROT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), retval)
	ph.broadcastInRoom(server, p, area, room, slot)
}

func (ph *PacketHandler) broadcastPlayerOK(server *ServerThread, socket net.Conn) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot

	// 0,0; 0,0
	playerok := []byte{0, 0, 0, 0}
	playerok[1] = cl.player
	p := NewPacket(commands.PLAYEROK, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), playerok)
	ph.broadcastInSlot(server, p, area, room, slotnr)
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
	slotstatus := []byte{0x00, 0x00, 0x00}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	// TODO: selecting eg area 5 causes this to output area 0 room 5.
	// not really clear what "area" is actually aligning with here.
	ph.debug("area: %d, room: %d, slot: %d\n", area, room, slotnr) // Debug print
	slotstatus[0] = byte(slotnr>>8) & 0xff
	slotstatus[1] = byte(slotnr) & 0xff
	slotstatus[2] = ph.slots.GetStatus(area, room, slotnr)
	p := NewPacket(commands.SLOTSTATUS, commands.TELL, commands.SERVER, ps.pid, slotstatus)
	ph.addOutPacket(server, socket, p)
}

// this is send when a client gets the rules but decides not to join
// also when a host decides not to create a gameslot
// AND when client or host leave the set gameslot
func (ph *PacketHandler) sendCancelSlot(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	ishost := cl.host

	// game creation is canceled
	// reset slot and leave it
	if ishost == 1 {
		cl.host = 0
		ph.slots.GetSlot(area, room, slotnr).Reset()
		ph.broadcastCancelSlot(server, area, room, slotnr)
		ph.broadcastPasswdProtect(server, area, room, slotnr)
		ph.broadcastSlotSceneType(server, area, room, slotnr)
		ph.broadcastSlotTitle(server, area, room, slotnr)
	}

	// normal players just leave
	ph.broadcastLeaveSlot(server, socket)
	cl.player = 0
	cl.slot = 0
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, area, room, 0)

	ph.broadcastSlotAttrib2(server, area, room, slotnr)

	// set status back to let others in
	n := ph.clients.CountPlayersInSlot(area, room, slotnr)
	if (n < int(ph.slots.GetMaximumPlayers(area, room, slotnr))) && ishost == 0 {
		ph.slots.GetSlot(area, room, slotnr).SetStatus(STATUS_GAMESET)
	}

	ph.broadcastSlotPlayerStatus(server, area, room, slotnr)
	ph.broadcastSlotStatus(server, area, room, slotnr)

	p := NewPacketWithoutPayload(commands.CANCELSLOT, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)

}

func (ph *PacketHandler) broadcastLeaveSlot(server *ServerThread, socket net.Conn) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot

	// 0,6; 0,0,0,0,0,0
	wholeaves := []byte{0, 6, 0, 0, 0, 0, 0, 0}
	who := cl.hnPair.handle

	copy(wholeaves[2:], who)
	p := NewPacket(commands.LEAVESLOT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), wholeaves)
	ph.broadcastInSlot(server, p, area, room, slotnr)

}

func (ph *PacketHandler) broadcastCancelSlot(server *ServerThread, area, room, slot int) {
	mess := NewPacketString("<LF=6><BODY><CENTER>host cancelled game<END>").GetData()
	p := NewPacket(commands.CANCELSLOTBC, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), mess)
	ph.broadcastInSlot(server, p, area, room, slot)
}

func (ph *PacketHandler) sendSlotPasswd(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	slot := ph.slots.GetSlot(area, room, slotnr)
	slot.SetPassword(ps.GetDecryptedString()) // TODO: apparently not GetDecryptedPassword()
	p := NewPacket(commands.SLOTPASSWD, commands.TELL, commands.SERVER, ps.pid, ps.pay)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendGetInfo(server *ServerThread, socket net.Conn, ps *Packet) {
	url := ps.GetDecryptedString()
	d := ph.information.GetData(string(url))

	mess := make([]byte, len(d)+len(url)+4)

	// putShort
	binary.BigEndian.PutUint16(mess[0:2], uint16(len(url)))
	// put
	copy(mess[2:2+len(url)], url)
	off := 2 + len(url)
	binary.BigEndian.PutUint16(mess[off:off+2], uint16(len(d)))
	copy(mess[off+2:], d)

	p := NewPacket(commands.GETINFO, commands.TELL, commands.SERVER, ps.pid, mess)
	ph.addOutPacket(server, socket, p)
}

// unknown, simply accept it
func (ph *PacketHandler) send6181(server *ServerThread, socket net.Conn, ps *Packet) {
	p := NewPacketWithoutPayload(commands.UNKN6181, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPlayerCount(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	playercount := []byte{0}
	playercount[0] = byte(ph.clients.CountPlayersInSlot(area, room, slotnr))
	p := NewPacket(commands.PLAYERCOUNT, commands.TELL, commands.SERVER, ps.pid, playercount)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPlayerNumber(server *ServerThread, socket net.Conn, ps *Packet) {
	num := []byte{0}
	num[0] = ph.clients.FindClientBySocket(socket).player
	p := NewPacket(commands.PLAYERNUMBER, commands.TELL, commands.SERVER, ps.pid, num)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPlayerStat(server *ServerThread, socket net.Conn, ps *Packet) {
	status := []byte{0, 0}
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	player := ps.pay[0] // query which player ?
	cl = ph.clients.FindClientBySlot(area, room, slotnr, int(player))

	if cl != nil {
		status = cl.GetPreGameStat(player)
	} else { // client left us :(
		status[0] = 0 // TODO: not sure if this will help
	}

	p := NewPacket(commands.PLAYERSTAT, commands.TELL, commands.SERVER, ps.pid, status)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPlayerScore(server *ServerThread, socket net.Conn, ps *Packet) {
	score := []byte{0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00}

	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot
	player := ps.pay[0]

	// TESTpacket. where can i see those ?
	r := make([]byte, (1+2)+(5*4))
	off := 0
	r[off] = player
	off++

	scenario := ph.slots.GetScenario(area, room, slotnr)
	binary.BigEndian.PutUint16(r[off:off+2], uint16(scenario))
	off += 2

	binary.BigEndian.PutUint32(r[off:off+4], uint32(110))
	off += 4

	binary.BigEndian.PutUint32(r[off:off+4], uint32(220))
	off += 4

	binary.BigEndian.PutUint32(r[off:off+4], uint32(330))
	off += 4

	binary.BigEndian.PutUint32(r[off:off+4], uint32(440))
	off += 4

	binary.BigEndian.PutUint32(r[off:off+4], uint32(550))
	off += 4

	copy(score, r)

	// TODO: send the scoring from ranklist for this player
	score[0] = player
	// score[1] = byte(scenario>>8) & 0xff
	// score[2] = byte(scenario) & 0xff
	p := NewPacket(commands.PLAYERSCORE, commands.TELL, commands.SERVER, ps.pid, score)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendGameSession(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	gameno := cl.GameNumber
	sess := fmt.Sprintf("%015d", gameno)

	buff := make([]byte, 19)
	off := 0
	binary.BigEndian.PutUint16(buff[off:off+2], uint16(0x0f))
	off += 2
	copy(buff[off:], []byte(sess))
	off += 15
	binary.BigEndian.PutUint16(buff[off:off+2], uint16(0x00))

	p := NewPacket(commands.GAMESESSION, commands.TELL, commands.SERVER, ps.pid, buff)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendDifficulty(server *ServerThread, socket net.Conn, ps *Packet) {
	difficulty := []byte{0x00, 0x10,
		0x01, 0x03, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	room := cl.room
	slotnr := cl.slot

	// TODO: here is sent more (different game modes). more tests!
	difficulty[3] = ph.slots.GetDifficulty(area, room, slotnr)
	difficulty[4] = ph.slots.GetFriendlyFire(area, room, slotnr)
	p := NewPacket(commands.GAMEDIFF, commands.TELL, commands.SERVER, ps.pid, difficulty)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendLogout(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	area := cl.area
	ph.db.UpdateClientGame(cl.userID, 0)
	ph.removeClient(server, cl)

	ph.broadcastAreaPlayerCnt(server, socket, area)
	p := NewPacketWithoutPayload(commands.LOGOUT, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendGSinfo(server *ServerThread, socket net.Conn, ps *Packet) {
	// 0x00,0x04; 192,168,7,77; 0x00,0x02; 0x21; 0xF2(port)
	// 0x00; 0x00; 0x1e; 0x00
	gsinfo := []byte{0x00, 0x04, byte(192), byte(168), byte(7), byte(77),
		0x00, 0x02, 0x21, byte(0xF2), // port 8690
		0x00, 0x00, 0x1e, 0x00}

	gsinfo[2] = ph.gsIP[0]
	gsinfo[3] = ph.gsIP[1]
	gsinfo[4] = ph.gsIP[2]
	gsinfo[5] = ph.gsIP[3]

	// todo: usage of multiple gameservers (why?)
	p := NewPacket(commands.GSINFO, commands.TELL, commands.SERVER, ps.pid, gsinfo)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendEnterAGL(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	gamenum, err := ph.db.GetGameNumber(cl.userID)
	if err != nil {
		ph.debug("Error getting game number: %s\n", err)
		return
	}
	cl.GameNumber = gamenum
	cl.area = 51
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, 51, cl.room, cl.slot)

	p := NewPacketWithoutPayload(commands.ENTERAGL, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)

	// broadcast new playercount
	ph.broadcastAglPlayerCnt(server, cl.GameNumber)

	status := cl.GetCharacterStat()
	p = NewPacket(commands.AGLJOIN, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), status)
	ph.broadcastInAgl(server, p, gamenum)

}

func (ph *PacketHandler) sendLeaveAGL(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	gamenum := cl.GameNumber

	// broadcast leaving of player
	wholeave := []byte{0, 6, 0, 0, 0, 0, 0, 0}
	who := cl.hnPair.handle
	copy(wholeave[2:], who)
	p := NewPacket(commands.LEAVEAGL, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), wholeave)
	ph.broadcastInAgl(server, p, gamenum)

	// set player back into area selection
	cl.area = 0
	cl.GameNumber = 0
	ph.db.UpdateClientGame(cl.userID, 0)
	ph.db.UpdateClientOrigin(cl.userID, STATUS_LOBBY, 0, 0, 0)

	p = NewPacketWithoutPayload(commands.LEAVEAGL, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)

	// broadcast new number of players in agl
	ph.broadcastAglPlayerCnt(server, gamenum)

	// TODO: this is an assumption, need to differentiate with multiple rooms
	ph.broadcastRoomPlayerCnt(server, 1, 1)
}

func (ph *PacketHandler) sendAGLstats(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	gamenum := cl.GameNumber
	aglstats := make([]byte, 1024)
	off := 0
	binary.BigEndian.PutUint16(aglstats[off:off+2], 0)
	off += 2
	aglstats[off] = byte(3) // unknown what this is
	off++
	aglstats[off] = byte(ph.clients.GetPlayerCountAgl(gamenum))
	off++
	for _, c := range ph.clients.GetList() {
		if c.GameNumber == gamenum {
			stats := c.GetCharacterStat()
			statlen := len(stats)
			copy(aglstats[off:], stats)
			off += statlen
		}
	}
	status := aglstats[:off]
	p := NewPacket(commands.AGLSTATS, commands.TELL, commands.SERVER, ps.pid, status)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendAGLplayerCnt(server *ServerThread, socket net.Conn, ps *Packet) {
	cnt := []byte{0, 1}
	cl := ph.clients.FindClientBySocket(socket)
	cnt[1] = byte(ph.clients.GetPlayerCountAgl(cl.GameNumber))
	p := NewPacket(commands.AGLPLAYERCNT, commands.TELL, commands.SERVER, ps.pid, cnt)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendEventDat(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)
	// area := cl.area
	// room := cl.room
	// slotnr := cl.slot
	// game := cl.GameNumber

	// 0,6; 0,0,0,0,0,0
	rcpthandle := []byte{0, 6, 0, 0, 0, 0, 0, 0}
	recpt := []byte{0, 0, 0, 0, 0, 0}
	event := ps.GetEventData()
	copy(rcpthandle[2:], event[2:8])
	copy(recpt, event[2:8])

	eventlen := (int(event[8]) << 8) + int(event[9])&0xff

	// create the event packet: sender, eventdat, and their lengths
	eventpacket := make([]byte, eventlen+2+6+2)
	off := 0
	binary.BigEndian.PutUint16(eventpacket[off:off+2], uint16(6))
	off += 2
	copy(eventpacket[off:], cl.hnPair.handle)
	off += 6
	binary.BigEndian.PutUint16(eventpacket[off:off+2], uint16(eventlen))
	off += 2
	copy(eventpacket[off:], event[10:10+eventlen])
	off += eventlen

	// p := NewPacket(commands.EVENTDATBC, commands.TELL, commands.SERVER, ps.pid, eventpacket)
	p := NewPacket(commands.EVENTDATBC, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), eventpacket)
	rcl := ph.clients.FindClientByHandle(string(recpt))
	if rcl != nil {
		ph.addOutPacket(server, rcl.socket, p)
	}

	// accept event data by sending back unencrypted recipient
	p = NewPacket(commands.EVENTDAT, commands.TELL, commands.SERVER, ps.pid, rcpthandle)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendBuddyList(server *ServerThread, socket net.Conn, ps *Packet) {
	var p *Packet
	offline := NewPacketString("<BODY><SIZE=3>not connected<END>").GetData()

	//0,0; 0,0; 0,0; 0
	online := []byte{0, 0, 0, 0, 0, 0, 0}
	ingame := []byte{0, 0, 0, 0, 0, 0, 1}

	handle := ps.GetDecryptedString()

	status := ph.clients.GetClientStatus(handle)

	switch status {
	case 1:
		p = NewPacket(commands.BUDDYLIST, commands.TELL, commands.SERVER, ps.pid, online)
	case 3:
		p = NewPacket(commands.BUDDYLIST, commands.TELL, commands.SERVER, ps.pid, ingame)
	default:
		p = NewPacket(commands.BUDDYLIST, commands.TELL, commands.SERVER, ps.pid, offline)
		p.SetErr()
	}

	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendCheckBuddy(server *ServerThread, socket net.Conn, ps *Packet) {
	var p *Packet
	offline := NewPacketString("<BODY><SIZE=3><CENTER>not connected<END>").GetData()
	online := []byte{
		0x00, 0x0c, 0x30, 0x61, 0x64, 0x36, 0x30, 0x31, 0x30, 0x38, 0x32, 0x30, 0x30, 0x38, // 0ad601082008
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
		0x00, 0x29, 0x3c, 0x42, 0x4f, 0x44, 0x59, 0x3e, // <BODY>
		0x3c, 0x53, 0x49, 0x5a, 0x45, 0x3d, 0x33, 0x3e, // <SIZE=3>
		0x82, 0x65, 0x82, 0x71, 0x82, 0x64,
		0x82, 0x64, 0x83, 0x47, 0x83, 0x8a,
		0x83, 0x41, 0x82, 0xc9, 0x82, 0xa2,
		0x82, 0xdc, 0x82, 0xb7,
		0x3c, 0x45, 0x4e, 0x44, 0x3e, // <END>
	}
	ingame := []byte{
		0x00, 0x2b,
		0x3c, 0x42, 0x4f, 0x44, 0x59, 0x3e,
		0x3c, 0x53, 0x49, 0x5a, 0x45, 0x3d, 0x33, 0x3e,
		0x8c, 0xbb, 0x8d, 0xdd,
		0x81, 0x41, 0x83, 0x51, 0x81, 0x5b, 0x83, 0x80,
		0x83, 0x76, 0x83, 0x8c, 0x83, 0x43, 0x92, 0x86,
		0x82, 0xc5, 0x82, 0xb7,
		0x3c, 0x45, 0x4e, 0x44, 0x3e,
	}
	handle := ps.GetDecryptedString()
	status := ph.clients.GetClientStatus(handle)
	switch status {
	case 1:
		p = NewPacket(commands.CHECKBUDDY, commands.TELL, commands.SERVER, ps.pid, online)
	case 3:
		p = NewPacket(commands.CHECKBUDDY, commands.TELL, commands.SERVER, ps.pid, ingame)
		p.SetErr()
	default:
		p = NewPacket(commands.CHECKBUDDY, commands.TELL, commands.SERVER, ps.pid, offline)
		p.SetErr()
	}
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) sendPrivateMsg(server *ServerThread, socket net.Conn, ps *Packet) {
	offline := NewPacketString("<BODY><SIZE=3>not connected<END>").GetData()
	var p *Packet
	cl := ph.clients.FindClientBySocket(socket)
	mess := ps.GetDecryptedPvtMess(cl)
	cl = ph.clients.FindClientByHandle(string(mess.Recipient))
	if cl != nil {
		broadcast := mess.GetPacketData()
		// Accept the message packet
		p = NewPacket(commands.PRIVATEMSG, commands.TELL, commands.SERVER, ps.pid, nil)
		ph.addOutPacket(server, socket, p)
		// Broadcast message to recipient
		p = NewPacket(commands.PRIVATEMSGBC, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), broadcast)
		ph.addOutPacket(server, socket, p)
	} else {
		// Tell sender that recipient is offline
		p = NewPacket(commands.PRIVATEMSG, commands.TELL, commands.SERVER, ps.pid, offline)
		p.SetErr()
		ph.addOutPacket(server, socket, p)
	}
}

func (ph *PacketHandler) sendTimeout(server *ServerThread, socket net.Conn, ps *Packet) {
	// set default timeout to 590124 seconds (~10K minutes): 0x9012C
	// other options: 0x708 = 30min
	// 				  0x258 = 10min
	timeout := []byte{0x00, 0x09, 0x01, 0x2C, 0x00, 0x00, 0x02, 0x58}
	p := NewPacket(commands.UNKN61A0, commands.TELL, commands.SERVER, ps.pid, timeout)
	ph.addOutPacket(server, socket, p)
}

func (ph *PacketHandler) send61A1(server *ServerThread, socket net.Conn, ps *Packet) {
	// not 100% sure what this does yet; presumed to be latency
	latency := []byte{0x00, 0x00, 0x03, byte(0x84), 0x00, 0x00, 0x07, 0x08, 0x00, 0x00}
	p := NewPacket(commands.UNKN61A1, commands.TELL, commands.SERVER, ps.pid, latency)
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

func (ph *PacketHandler) broadcastAglPlayerCnt(server *ServerThread, gamenr int) {
	cnt := []byte{0, 1}
	cnt[1] = byte(ph.clients.GetPlayerCountAgl(gamenr))
	p := NewPacket(commands.AGLPLAYERCNT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), cnt)
	ph.broadcastInAgl(server, p, gamenr)
}

func (ph *PacketHandler) broadcastInAgl(server *ServerThread, p *Packet, gamenr int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.GameNumber == gamenr && cl.GameNumber > 0 {
			ph.addOutPacket(server, cl.socket, p)
		}
	}
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

func (ph *PacketHandler) broadcastInRoom(server *ServerThread, p *Packet, area, room, slot int) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		if cl.area == area && cl.room == room && cl.slot == slot {
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

// send this every 30 seconds to all clients; handled by heartbeatthread
// TODO: what does the payload mean ?
func (ph *PacketHandler) BroadcastPing(server *ServerThread) {
	data := []byte{0x00, 0x02, 0x00, 0x01, 0x03, byte(0xe7), 0x00, 0x01}
	p := NewPacket(commands.HEARTBEAT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), data)
	ph.broadcastPacket(server, p)
}

// broadcast a packet to all connected clients
func (ph *PacketHandler) broadcastPacket(server *ServerThread, p *Packet) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		ph.addOutPacket(server, cl.socket, p)
	}
}

func (ph *PacketHandler) removeClient(server *ServerThread, cl *Client) {
	if cl == nil {
		return
	}
	ph.debug("Removing client %s\n", cl.userID)
	// // If needed, lock the client (e.g., cl.mu.Lock(); defer cl.mu.Unlock())
	cl.mu.Lock()
	defer cl.mu.Unlock()
	area := cl.area
	room := cl.room
	slot := cl.slot
	// game := cl.gamenumber
	socket := cl.socket
	host := cl.host
	who := cl.hnPair.handle

	// Set the client status to offline.
	if err := ph.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0); err != nil {
		fmt.Println("Error updating client origin to offline:", err)
	}

	// Remove the client from the list.
	ph.clients.Remove(cl)

	// If the client was a host and occupying a slot, perform slot-specific broadcasts.
	if host == 1 && slot != 0 {
		ph.slots.GetSlot(area, room, slot).Reset()
		ph.broadcastCancelSlot(server, area, room, slot)
		ph.broadcastPasswdProtect(server, area, room, slot)
		ph.broadcastSlotSceneType(server, area, room, slot)
		ph.broadcastSlotTitle(server, area, room, slot)
		ph.broadcastSlotAttrib2(server, area, room, slot)
		ph.broadcastSlotPlayerStatus(server, area, room, slot)
		ph.broadcastSlotStatus(server, area, room, slot)
	}

	// If the client was not a host but still in a slot.
	if slot != 0 && host == 0 {
		// Prepare a broadcast packet to notify other players in the slot.
		wholeaves := []byte{0, 6, 0, 0, 0, 0, 0, 0}
		copy(wholeaves[2:], who)
		p := NewPacket(commands.LEAVESLOT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), wholeaves)
		ph.broadcastInSlot(server, p, area, room, slot)

		// If there is room for additional players and a host is still present, update slot status.
		n := ph.clients.CountPlayersInSlot(area, room, slot)
		maxPlayers := int(ph.slots.GetMaximumPlayers(area, room, slot))
		if n < maxPlayers {
			if ph.clients.GetHostOfSlot(area, room, slot) != nil {
				ph.slots.GetSlot(area, room, slot).SetStatus(STATUS_GAMESET)
			}
		}

		// If this was the last client in the slot, reset the slot and broadcast related changes.
		if ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
			ph.slots.GetSlot(area, room, slot).Reset()
			ph.broadcastPasswdProtect(server, area, room, slot)
			ph.broadcastSlotSceneType(server, area, room, slot)
			ph.broadcastSlotTitle(server, area, room, slot)
		}

		ph.broadcastSlotAttrib2(server, area, room, slot)
		ph.broadcastSlotPlayerStatus(server, area, room, slot)
		ph.broadcastSlotStatus(server, area, room, slot)
	}

	// // In the after-game lobby (area 51) with a valid game number, you might need extra handling.
	// if area == 51 && game != 0 {
	// 	// TODO: is this really necessary?
	// }

	// Broadcast the updated room player count.
	ph.broadcastRoomPlayerCnt(server, area, room)

	// Finally, disconnect the client socket.
	server.Disconnect(socket)
}

func (ph *PacketHandler) send6002(server *ServerThread, socket net.Conn, ps *Packet) {
	cl := ph.clients.FindClientBySocket(socket)

	area := cl.area
	room := cl.room
	slot := cl.slot

	// reset client's area/slot

	cl.area = 0
	cl.room = 0
	cl.slot = 0
	cl.player = 0

	//free slot for other players when last player left
	// need to implement theese:
	if ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
		ph.slots.GetSlot(area, room, slot).Reset()
		ph.broadcastSlotPlayerStatus(server, area, room, slot)
		ph.broadcastPasswdProtect(server, area, room, slot)
		ph.broadcastSlotTitle(server, area, room, slot)
		ph.broadcastSlotSceneType(server, area, room, slot)
		ph.broadcastSlotAttrib2(server, area, room, slot)
		ph.broadcastSlotStatus(server, area, room, slot)
	}

	p := NewPacketWithoutPayload(commands.UNKN6002, commands.TELL, commands.SERVER, ps.pid)
	ph.addOutPacket(server, socket, p)
}

// send a query to every client
// the answer sets back the alive flag
// if this doesn't happen the client is deleted from list
func (ph *PacketHandler) BroadcastConnCheck(server *ServerThread) {
	p := NewPacketWithoutPayload(commands.CONNCHECK, commands.QUERY, commands.SERVER, ph.getNextPacketID())
	for _, cl := range ph.clients.GetList() {
		if cl.area != 51 {
			if cl.ConnAlive {
				cl.ConnAlive = false
				ph.addOutPacket(server, cl.socket, p)
			} else {
				ph.debug("Client %s did not respond to CONNCHECK; it's OUTTA HERE\n", cl.userID)
				ph.removeClient(server, cl)
			}
		}
	}
}

func (ph *PacketHandler) RemoveClientNoDisconnect(server *ServerThread, socket net.Conn) {
	cl := ph.clients.FindClientBySocket(socket)

	if cl == nil {
		return
	}
	cl.ConnAlive = false

	ph.debug("client: %s socket: %p\n", cl.userID, socket)
	// locking becauset his function can be called by both
	// server and handlerthread
	cl.mu.Lock()
	defer cl.mu.Unlock()
	area := cl.area
	room := cl.room
	slot := cl.slot
	// game := cl.GameNumber
	host := cl.host
	who := cl.hnPair.handle

	// Set the client status to offline.
	if err := ph.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0); err != nil {
		ph.debug("Error updating client origin to offline: %v\n", err)
	}

	// Remove the client from the list.
	ph.clients.Remove(cl)
	ph.debug("Client %s removed but kept session alive\n", cl.userID)

	// If the client was a host and occupying a slot, perform slot-specific broadcasts.
	if host == 1 && slot != 0 {
		ph.slots.GetSlot(area, room, slot).Reset()
		ph.broadcastCancelSlot(server, area, room, slot)
		ph.broadcastPasswdProtect(server, area, room, slot)
		ph.broadcastSlotSceneType(server, area, room, slot)
		ph.broadcastSlotTitle(server, area, room, slot)
		ph.broadcastSlotAttrib2(server, area, room, slot)
		ph.broadcastSlotPlayerStatus(server, area, room, slot)
		ph.broadcastSlotStatus(server, area, room, slot)
	}

	// If the client was not a host but still in a slot.
	if slot != 0 && host == 0 {
		// Prepare a broadcast packet to notify other players in the slot.
		wholeaves := []byte{0, 6, 0, 0, 0, 0, 0, 0}
		copy(wholeaves[2:], who)
		p := NewPacket(commands.LEAVESLOT, commands.BROADCAST, commands.SERVER, ph.getNextPacketID(), wholeaves)
		ph.broadcastInSlot(server, p, area, room, slot)

		// If there is room for additional players and a host is still present, update slot status.
		n := ph.clients.CountPlayersInSlot(area, room, slot)
		maxPlayers := int(ph.slots.GetMaximumPlayers(area, room, slot))
		if n < maxPlayers {
			if ph.clients.GetHostOfSlot(area, room, slot) != nil {
				ph.slots.GetSlot(area, room, slot).SetStatus(STATUS_GAMESET)
			}
		}

		// If this was the last client in the slot, reset the slot and broadcast related changes.
		if ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
			ph.slots.GetSlot(area, room, slot).Reset()
			ph.broadcastPasswdProtect(server, area, room, slot)
			ph.broadcastSlotSceneType(server, area, room, slot)
			ph.broadcastSlotTitle(server, area, room, slot)
		}

		ph.broadcastSlotAttrib2(server, area, room, slot)
		ph.broadcastSlotPlayerStatus(server, area, room, slot)
		ph.broadcastSlotStatus(server, area, room, slot)
	}

	// // In the after-game lobby (area 51) with a valid game number, you might need extra handling.
	// if area == 51 && game != 0 {
	// 	// TODO: is this really necessary?
	// }

	// Broadcast the updated room player count.
	ph.broadcastRoomPlayerCnt(server, area, room)
}

// check the livetime of the slot
// broadcast the autostart on zero
// used for the East Town (area 0x001)
func (ph *PacketHandler) CheckAutoStart(server *ServerThread) {
	cls := ph.clients.GetList()
	for _, cl := range cls {
		area := cl.area
		room := cl.room
		slot := cl.slot
		if (area == 1 && room == 1 && slot != 0) {
			livetime := ph.slots.GetSlot(area, room, slot).GetLivetime()
			ph.debug("Livetime of area 1 room 1: %d\n", livetime)
			if livetime == 0 {
				ph.broadcastGetReady(server, cl.socket)
			}
		}
	}
}

// helper for keeping slots open when something went wrong
func (ph *PacketHandler) CleanGhostRooms(server *ServerThread) {
	// 1. we should probably just use .Reset() on the slot instead of simply changing its status
	// 2. i think slot is 0-indexed elsewhere in the code. why is it 1-indexed here? ...
	for area := 1; area <= ph.areas.GetAreaCount(); area++ {
		for room := 1; room <= ph.rooms.GetRoomCount(); room++ {
			for slot := 1; slot <= 20; slot++ {
				if ph.slots.GetStatus(area, room, slot) == STATUS_GAMESET && ph.clients.CountPlayersInSlot(area, room, slot) == 0 {
					// ph.slots.GetSlot(area, room, slot).Reset()
					ph.slots.GetSlot(area, room, slot).SetStatus(STATUS_FREE)
					ph.broadcastSlotStatus(server, area, room, slot)
					ph.debug("Cleaned ghost room: a%d r%d s%d\n", area, room, slot)
				}
			}
		}
	}

}