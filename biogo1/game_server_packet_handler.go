package main

import (
	"fmt"
	"main/commands"
	"net"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type GameServerDataEvent struct {
	server *GameServerThread
	conn   net.Conn
	data   []byte
}

type GameServerPacketHandler struct {
	clients         *ClientList
	db              *Database
	packetidcounter int
	queue           chan GameServerDataEvent
	logger          *log.Logger
}

func NewGameServerPacketHandler() *GameServerPacketHandler {
	db, err := NewDatabase("bioserver", "xxxxxxxxxxxxxxxx")
	if err != nil {
		fmt.Println("NewGameServerPacketHandler() Error opening database connection:", err)
		return nil
	}

	return &GameServerPacketHandler{
		clients:         NewClientList(),
		db:              db,
		packetidcounter: 0,
		queue:           make(chan GameServerDataEvent, 100), // buffered channel
		logger:          log.New(os.Stdout, "", log.Ltime),
	}
}

func (gsp *GameServerPacketHandler) debug(format string, v ...interface{}) {
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

	gsp.logger.Printf("%s:%d %s() %s", file, line, funcName, msg)
}


// func (gsp *GameServerPacketHandler) debug(format string, a ...interface{}) {
// 	fmt.Printf(format, a...)
// }

func (gsp *GameServerPacketHandler) Run() {
	fmt.Println("GameServerPacketHandler started")

	for dataEvent := range gsp.queue {
		// Process the data event
		gsp.handleDataEvent(dataEvent)
	}

	// for {
	//     select {
	//     case dataEvent := <-gsp.queue:
	//         // Process the data event
	//         gsp.handleDataEvent(dataEvent)
	//     }
	// }

}

func (gsp *GameServerPacketHandler) handleDataEvent(dataEvent GameServerDataEvent) {
	// send the data
	dataEvent.server.send(dataEvent.conn, dataEvent.data)
}

func (gsp *GameServerPacketHandler) ProcessData(server *GameServerThread, conn net.Conn, data []byte, length int) {
	switch data[0] {
	case 0x82:
		if data[1] == 0x02 {
			// some session checking etc.
			p := NewPacketFromBytes(data)
			if p.cmd == commands.GSLOGIN {
				// check this session and if ok create client
				if !gsp.checkSession(server, conn, p) {
					gsp.debug("Session check failed for %s\n", conn.RemoteAddr().String())
				}
			}
		}
	default:
		// broadcast to connected clients in gamesession but not the sender
		acopy := make([]byte, length)
		copy(acopy, data)

		cl := gsp.clients.FindClientBySocket(conn)
		cl.ConnAlive = true
		gamenum := cl.GameNumber
		cls := gsp.clients.GetList()
		for _, client := range cls {
			if client.GameNumber == gamenum && client.socket != conn {
				gsp.queue <- GameServerDataEvent{server, client.socket, acopy}
			}
		}
	}
}

func (gsp *GameServerPacketHandler) AddOutPacket(server *GameServerThread, conn net.Conn, packet *Packet) {
	// TODO: LOGGING
	gsp.queue <- GameServerDataEvent{server, conn, packet.GetPacketData()}
}

func (gsp *GameServerPacketHandler) BroadcastPacket(server *GameServerThread, packet *Packet) {
	cls := gsp.clients.GetList()
	for _, client := range cls {
		gsp.queue <- GameServerDataEvent{server, client.socket, packet.GetPacketData()}
	}
}

func (gsp *GameServerPacketHandler) CountInGamePlayers() int {
	return len(gsp.clients.GetList())
}

func (gsp *GameServerPacketHandler) getClients() *ClientList {
	return gsp.clients
}

func (gsp *GameServerPacketHandler) getNextPacketID() int {
	gsp.packetidcounter++
	return gsp.packetidcounter
}

func (gsp *GameServerPacketHandler) GSsendLogin(server *GameServerThread, conn net.Conn) {
	p := NewPacketWithoutPayload(commands.GSLOGIN, commands.QUERY, commands.GAMESERVER, gsp.getNextPacketID())
	gsp.AddOutPacket(server, conn, p)
}

func (gsp *GameServerPacketHandler) checkSession(server *GameServerThread, socket net.Conn, ps *Packet) bool {
	seed := ps.pid

	sessA := int(ps.pay[0]-0x30) * 10000
	sessA += int(ps.pay[1]-0x30) * 1000
	sessA += int(ps.pay[2]-0x30) * 100
	sessA += int(ps.pay[3]-0x30) * 10
	sessA += int(ps.pay[4] - 0x30)

	sessB := int(ps.pay[5]-0x30) * 10000
	sessB += int(ps.pay[6]-0x30) * 1000
	sessB += int(ps.pay[7]-0x30) * 100
	sessB += int(ps.pay[8]-0x30) * 10
	sessB += int(ps.pay[9] - 0x30)

	session := fmt.Sprintf("%04d%04d", sessA-seed, sessB-seed)

	userid, err := gsp.db.GetUserID(session)
	if err != nil {
		gsp.debug("PacketHandler checkSession() Error getting user id:%v\n", err)
		return false
	}

	gsp.debug("Session: %s with UserID: %s\n", session, userid)

	if userid != "" {
		// loop through clients and remove old connections
		// then setup client object for this user/session
		cl := gsp.clients.FindClientByUserID(userid)
		if cl != nil {
			gsp.debug("Found a client with UserID %s: %v\n", userid, cl)
			for _, c := range gsp.clients.GetList() {
				if c != nil && c.userID == userid {
					gsp.debug("removing client with UserID %s & socket %p\n", userid, c.socket)
					gsp.removeClient(server, c)
				}
			}
		}

		gsp.clients.Add(NewClient(socket, userid, session))
		cl = gsp.clients.FindClientBySocket(socket)
		if cl == nil {
			gsp.debug("\n\n\nFailed to add client to client list!!! big problem!!!\n\n\n")
			return false
		}

		err = gsp.db.UpdateClientOrigin(userid, STATUS_LOBBY, 0, 0, 0)
		if err != nil {
			gsp.debug("Error updating client origin: %v\n", err)
			return false
		}

		gamenr, err := gsp.db.GetGameNumber(cl.userID)
		if err != nil {
			gsp.debug("Error getting game number: %v\n", err)
			return false
		}
		cl.GameNumber = gamenr

		gsp.db.UpdateClientOrigin(cl.userID, STATUS_GAME, 0, 0, 0)
		return true
	} else {
		// session check failed; disconnect this client
		gsp.debug("Session check failed!")
		server.disconnect(socket)
		return false
	}
}

func (gsp *GameServerPacketHandler) removeClient(server *GameServerThread, cl *Client) {
	if cl == nil {
		gsp.debug("removeClient() called with nil client\n")
		return
	}

	sock := cl.socket
	gsp.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0)
	gsp.clients.Remove(cl)
	server.disconnect(sock)
}

func (gsp *GameServerPacketHandler) removeClientByID(server *GameServerThread, userid string) {
	cl := gsp.clients.FindClientByUserID(userid)
	if cl != nil {
		gsp.removeClient(server, cl)
	}
}

func (gsp *GameServerPacketHandler) RemoveClientNoDisconnect(server *GameServerThread, conn net.Conn) {
	cl := gsp.clients.FindClientBySocket(conn)
	// set user to offline status in database
	if cl != nil {
		gsp.db.UpdateClientOrigin(cl.userID, STATUS_OFFLINE, -1, 0, 0)
		gsp.clients.Remove(cl)
	}
}

func (gsp *GameServerPacketHandler) ConnCheck(server *GameServerThread) {
	cls := gsp.clients.GetList()
	for _, cl := range cls {
		if cl == nil {
			continue
		}
		if cl.ConnAlive {
			cl.ConnAlive = false
		} else { 
			gsp.removeClient(server, cl)
		}
	}
}