package main

import "encoding/binary"

type PrivateMessage struct {
	SenderHandle []byte
	SenderName   []byte
	Recipient    []byte
	Message      []byte
}

func NewPrivateMessage(senderHandle, senderName, recipient, message []byte) *PrivateMessage {
	return &PrivateMessage{
		SenderHandle: senderHandle,
		SenderName:   senderName,
		Recipient:    recipient,
		Message:      message,
	}
}

func (pm *PrivateMessage) GetPacketData() []byte {
	z := make([]byte, 200)
	off := 0

	binary.BigEndian.PutUint16(z[off:off+2], uint16(len(pm.SenderHandle)))
	off += 2
	copy(z[off:off+len(pm.SenderHandle)], pm.SenderHandle)
	off += len(pm.SenderHandle)

	binary.BigEndian.PutUint16(z[off:off+2], uint16(len(pm.SenderName)))
	off += 2
	copy(z[off:off+len(pm.SenderName)], pm.SenderName)
	off += len(pm.SenderName)

	binary.BigEndian.PutUint16(z[off:off+2], uint16(len(pm.Message)))
	off += 2
	copy(z[off:off+len(pm.Message)], pm.Message)
	off += len(pm.Message)

	retval := make([]byte, off)
	copy(retval, z[:off])
	return retval
}
