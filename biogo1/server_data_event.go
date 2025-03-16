package main

import (
	"net"
)

type ServerDataEvent struct {
	server *ServerThread
	socket net.Conn
	data []byte
}

func NewServerDataEvent(server *ServerThread, socket net.Conn, data []byte) *ServerDataEvent {
	return &ServerDataEvent{
		server: server,
		socket: socket,
		data: data,
	}
}
