package main

import (
	"encoding/binary"
	"net"
)

type Client struct {
	socket         net.Conn
	userID         string
	session        string
	characterStats []byte //0xd0 in len; TODO: make this a struct?
	character      int16
	costume        int16
	area           int //special case 51 = post-game lobby
	room           int
	slot           int
	GameNumber     int
	player         byte    // number of this player (1-4)
	ConnAlive      bool    // set back every 60sec or be disconnected
	host           byte    // host of a gameslot
	hnPair         *HNPair //chosen handle/nickname
}

func NewClient(socket net.Conn, userID string, session string) *Client {
	return &Client{
		socket:    socket,
		userID:    userID,
		session:   session,
		area:      0, //no area (area selection screen)
		room:      0, //no room
		slot:      0, //no slot
		host:      0,
		ConnAlive: true,
	}
}

func (c *Client) GetPreGameStat(playernum byte) []byte {
	z := make([]byte, 300)
	off := 0
	z[off] = playernum
	off++
	z[off] = 1
	off++

	hnpair := c.hnPair.GetHNPair() // GetHNPair returns []byte, per the Java version.
	copy(z[off:], hnpair)
	off += len(hnpair)

	binary.BigEndian.PutUint16(z[off:off+2], uint16(len(c.characterStats)))
	off += 2

	copy(z[off:], c.characterStats)
	off += len(c.characterStats)

	z[off] = 0
	off++
	z[off] = 0
	off++
	z[off] = 6
	off++
	return z[:off]
}