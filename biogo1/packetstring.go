package main

import (
	"encoding/binary"
)

type PacketString struct {
	buffer []byte
}

func NewPacketString(s string) *PacketString {
	return &PacketString{buffer: []byte(s)}
}

// GetData returns the byte array representation used in packets
// Format: 2 bytes for length (short) followed by the actual data
func (ps *PacketString) GetData() []byte {
	// Create a new byte array with length of buffer + 2 bytes for length field
	result := make([]byte, len(ps.buffer)+2)

	// Write the length of the buffer as a short (2 bytes) at the beginning
	binary.BigEndian.PutUint16(result[:2], uint16(len(ps.buffer)))

	// Copy the buffer bytes after the length field
	copy(result[2:], ps.buffer)

	return result
}
