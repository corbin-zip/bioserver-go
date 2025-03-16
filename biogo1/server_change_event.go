package main

import (
	"net"
)

const (
	REGISTER   = 1
	CHANGEOPS  = 2
	FORCECLOSE = 3
)

type ServerChangeEvent struct {
	conn      net.Conn
	EventType int
	ops	  int
}

func NewServerChangeEvent(conn net.Conn, eventType int, ops int) *ServerChangeEvent {
	return &ServerChangeEvent{
		conn:      conn,
		EventType: eventType,
		ops:	   ops,
	}
}
