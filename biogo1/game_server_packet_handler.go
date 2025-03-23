package main

import (
    "fmt"
    "net"
	"main/commands"
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
    }
}

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
                    fmt.Println("Session check gameserver failed!")
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
		// ph.debug("PacketHandler checkSession() Error getting user id:%v\n", err)
		return false
	}

	// ph.debug("Session: %s with UserID: %s\n", session, userid)

	if userid != "" {
		// loop through clients and remove old connections
		// then setup client object for this user/session
		cl := gsp.clients.FindClientByUserID(userid)
		if cl != nil {
			// ph.debug("Found a client with UserID %s: %v\n", userid, cl)
			for _, c := range gsp.clients.GetList() {
				if c.userID == userid {
					// ph.debug("removing client with UserID %s & socket %p\n", userid, c.socket)
					gsp.removeClient(server, c)
				}
			}
		}

		gsp.clients.Add(NewClient(socket, userid, session))
		cl = gsp.clients.FindClientBySocket(socket)
		if cl == nil {
			// fmt.Println("HandleInPacket checkSession() Failed to add client to client list!!! big problem!!!")
			return false
		}

		err = gsp.db.UpdateClientOrigin(userid, STATUS_LOBBY, 0, 0, 0)
		if err != nil {
			// fmt.Println("HandleInPacket checkSession() Error updating client origin:", err)
			return false
		}

		gamenr, err := gsp.db.GetGameNumber(cl.userID)
		if err != nil {
			// fmt.Println("HandleInPacket checkSession() Error getting game number:", err)
			return false
		}
		cl.GameNumber = gamenr

		gsp.db.UpdateClientOrigin(cl.userID, STATUS_GAME, 0, 0, 0)
		return true
	} else {
		// session check failed; disconnect this client
		// fmt.Println("HandleInPacket checkSession() Session check failed!")
		fmt.Println("Session check failed!")
		server.disconnect(socket)
		return false
	}
}

func (gsp *GameServerPacketHandler) removeClient(server *GameServerThread, cl *Client) {
	if cl == nil {
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