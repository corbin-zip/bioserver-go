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