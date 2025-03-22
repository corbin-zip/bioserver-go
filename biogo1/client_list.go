package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"slices"
)

type ClientList struct {
	clients []*Client
}

func NewClientList() *ClientList {
	return &ClientList{clients: make([]*Client, 0)}
}

func (cl *ClientList) Add(c *Client) {
	cl.clients = append(cl.clients, c)
}

func (cl *ClientList) GetList() []*Client {
	return cl.clients
}

func (cl *ClientList) FindClientBySocket(socket net.Conn) *Client {
	for _, c := range cl.clients {
		if c.socket == socket {
			return c
		}
	}
	return nil
}

func (cl *ClientList) FindClientByHandle(handle string) *Client {
	for _, c := range cl.clients {
		if string(c.hnPair.handle) == handle {
			return c
		}
	}
	return nil
}

func (cl *ClientList) FindClientByUserID(userid string) *Client {
	for _, c := range cl.clients {
		if c.userID == userid {
			return c
		}
	}
	return nil
}

func (cl *ClientList) FindClientBySlot(area, room, slot, player int) *Client {
	for _, c := range cl.clients {
		if c.area == area && c.room == room && c.slot == slot && c.player == byte(player) {
			return c
		}
	}
	return nil
}

func (cl *ClientList) Remove(c *Client) {
	for i, client := range cl.clients {
		if client == c {
			cl.clients = slices.Delete(cl.clients, i, i+1)
			break
		}
	}
}

func (cl *ClientList) CountPlayersInSlot(area, room, slot int) int {
	count := 0
	for _, c := range cl.clients {
		if c.slot == slot && c.area == area && c.room == room {
			count++
		}
	}
	return count
}

func (cl *ClientList) CountPlayersInArea(nr int) []int {
	// TODO: what is unknown 3rd value? is it ingame?
	retval := []int{0, 0, 0}

	for _, c := range cl.clients {
		if c.area == nr {
			if c.room == 0 {
				retval[0]++
			} else {
				retval[1]++
			}
		} else if c.area == 51 {
			retval[2]++
		}
	}
	return retval
}

func (cl *ClientList) CountPlayersInRoom(area int, room int) int {
	count := 0
	for _, c := range cl.clients {
		if c.room == room && c.area == area {
			count++
		}
	}
	return count
}

func (cl *ClientList) GetPlayerStats(area, room, slotnr int) []byte {
	retval := make([]byte, 1024)
	playercnt := byte(cl.CountPlayersInSlot(area, room, slotnr) & 0xff)

	// Create a buffer to write the data
	buffer := bytes.NewBuffer(retval[:0])

	// Write slotnr as a short (2 bytes)
	binary.Write(buffer, binary.BigEndian, uint16(slotnr))

	// Write the constant byte 3; TODO why ???
	buffer.WriteByte(3)

	// Write the player count
	buffer.WriteByte(playercnt)

	// Iterate over clients and add their stats to the buffer
	for _, client := range cl.clients {
		if client.area == area && client.room == room && client.slot == slotnr {
			buffer.Write(client.hnPair.GetHNPair())
			characterStats := client.characterStats
			binary.Write(buffer, binary.BigEndian, uint16(len(characterStats)))
			buffer.Write(characterStats)
		}
	}

	// Return the slice of the buffer's bytes
	return buffer.Bytes()
}

func (cl *ClientList) GetFreePlayerNum(area, room, slot int) int {
	fpn := []byte{0, 0, 0, 0, 0}
	for _, c := range cl.clients {
		if c.area == area && c.room == room && c.slot == slot {
			fpn[c.player] = 1
		}
	}
	for i := 2; i < 5; i++ {
		if fpn[i] == 0 {
			return i
		}
	}
	return 0
}

func (cl *ClientList) GetPlayerCountAgl(nr int) byte {
	count := byte(0)
	for _, c := range cl.clients {
		if c.GameNumber == nr {
			count++
		}
	}
	return count
}