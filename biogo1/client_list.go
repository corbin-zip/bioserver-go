package main

import (
	"fmt"
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
	fmt.Printf("Adding client %s\n", c.userID)
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

func (cl *ClientList) FindClientByUserID(userid string) *Client {
	for _, c := range cl.clients {
		if c.userID == userid {
			return c
		}
	}
	return nil
}

func (cl *ClientList) Remove(c *Client) {
	for i, client := range cl.clients {
		if client == c {
			cl.clients = slices.Delete(cl.clients, i, i + 1)
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