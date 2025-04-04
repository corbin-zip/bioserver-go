package main

import (
	"encoding/binary"
	// "fmt"
)

const (
	HEADER_SIZE = 12
)

type Packet struct {
	who byte   // server or client packet
	qsw byte   // question or answer
	cmd int    // command
	len int    // length including header
	pid int    // packet id
	err byte   // error
	pay []byte // payload
}

func NewPacket(command int, questionanswer byte, whosends byte, packetid int, payload []byte) *Packet {
	return &Packet{
		who: whosends,
		qsw: questionanswer,
		cmd: command,
		// len: HEADER_SIZE + len(payload),
		len: len(payload),
		pid: packetid,
		err: 0,
		pay: payload,
	}
}

func NewPacketWithoutPayload(command int, questionanswer byte, whosends byte, packetid int) *Packet {
	return &Packet{
		who: whosends,
		qsw: questionanswer,
		cmd: command,
		len: 0,
		pid: packetid,
		err: 0,
		pay: nil,
	}
}

func NewPacketFromBytes(data []byte) *Packet {
	return &Packet{
		who: data[0],
		qsw: data[1],
		cmd: int(data[2])<<8 | int(data[3]),
		len: int(data[4])<<8 | int(data[5]),
		pid: int(data[6])<<8 | int(data[7]),
		err: data[8],
		pay: data[HEADER_SIZE:],
	}
}

func (p *Packet) GetPacketData() []byte {
	result := make([]byte, p.len+HEADER_SIZE)

	// build the header
	result[0] = p.who
	result[1] = p.qsw
	result[2] = byte((p.cmd >> 8) & 0xff)
	result[3] = byte(p.cmd & 0xff)
	result[4] = byte((p.len >> 8) & 0xff)
	result[5] = byte(p.len & 0xff)
	result[6] = byte((p.pid >> 8) & 0xff)
	result[7] = byte(p.pid & 0xff)
	result[8] = byte(p.err)
	result[9] = 0xff
	result[10] = 0xff
	result[11] = 0xff

	if p.len > 0 {
		copy(result[HEADER_SIZE:], p.pay)
	}
	return result
}

func (p *Packet) CryptString() {
	// length := int(p.pay[0]) << 8 | int(p.pay[1]) - 2; // skip the sum
	length := ((int(p.pay[0]) << 8) | int(p.pay[1])) - 2 // skip the sum

	for i := 0; i < length; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}
}

func (p *Packet) GetVersion() []byte {
	length := ((int(p.pay[3]) << 8) | int(p.pay[4])) - 2 // skip the sum
	for i := 0; i < length; i++ {
		p.pay[7+i] = byte(p.pay[7+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	retval := make([]byte, length)
	copy(retval, p.pay[7:7+length])
	return retval
}

func (p *Packet) calcShift(i byte, pb byte) byte {
	fixval := []byte{21, 23, 10, 17, 23, 19, 6, 13}
	masks := []byte{0x33, 0x30, 0x3c, 0x34, 0x2d, 0x30, 0x3c, 0x34}
	return byte(fixval[i&7] - (i & 0xf8) - pb + ((pb-9+i)&masks[i&7])*2)
}

// decrypt chosen handle/nickname
func (p *Packet) GetDecryptedHNPair() *HNPair {
	hlen := ((int(p.pay[0]) << 8) | int(p.pay[1])) - 2           // skip the sum
	nlen := ((int(p.pay[hlen+4]) << 8) | int(p.pay[hlen+5])) - 2 // skip the sum

	for i := 0; i < hlen; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	for i := 0; i < nlen; i++ {
		p.pay[hlen+8+i] = byte(p.pay[hlen+8+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	handle := make([]byte, hlen)
	nickname := make([]byte, nlen)

	copy(handle, p.pay[4:4+hlen])
	copy(nickname, p.pay[hlen+8:hlen+8+nlen])

	return (NewHNPairFromBytes(handle, nickname))

}

// return first 2 bytes of payload as int
func (p *Packet) GetNumber() int {
	// TODO: i think this is causing me issues here....
	// essentially, this function is occasionally used to return the
	// specific area that we're in, and we're seeing that East Town is
	// returning 0 but i'm pretty sure it should be returning 1

	// return int(uint16((p.pay[0])<<8) | uint16(p.pay[1]))
	retval := ((int(p.pay[0]) << 8) & 0xFF00) | (int(p.pay[1]) & 0xFF)
	// fmt.Printf("\n\n!!!!!!!!\n\np.pay[0] is %d, p.pay[1] is %d; returning %d\n\n!!!!!!!!!!\n\n", p.pay[0], p.pay[1], retval)
	return retval
}

func (p *Packet) GetDecryptedString() []byte {
	length := ((int(p.pay[0]) << 8) | int(p.pay[1])) - 2 // skip the sum
	for i := 0; i < length; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	retval := make([]byte, length)
	copy(retval, p.pay[4:4+length])
	return retval
}

func (p *Packet) SetErr() {
	p.err = 0xff
}

func (p *Packet) GetPassword() []byte {
	length := ((int(p.pay[2]) << 8) | int(p.pay[3])) - 2 // skip the sum
	for i := 0; i < length; i++ {
		p.pay[6+i] = byte(p.pay[6+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	retval := make([]byte, length)
	copy(retval, p.pay[6:6+length])
	return retval
}

func (p *Packet) GetEventData() []byte {
	hlen := ((int(p.pay[0]) << 8) | int(p.pay[1])) - 2           // skip the sum
	elen := ((int(p.pay[hlen+4]) << 8) | int(p.pay[hlen+5])) - 2 // skip the sum

	for i := 0; i < hlen; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	for i := 0; i < elen; i++ {
		p.pay[hlen+8+i] = byte(p.pay[hlen+8+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	z := make([]byte, hlen+elen+4)
	off := 0

	binary.BigEndian.PutUint16(z[off:], uint16(hlen))
	off += 2
	copy(z[off:], p.pay[4:4+hlen])
	off += hlen
	binary.BigEndian.PutUint16(z[off:], uint16(elen))
	off += 2
	copy(z[off:], p.pay[hlen+8:hlen+8+elen])
	off += elen

	retval := make([]byte, off)
	copy(retval, z[:off])
	return retval
}

// decrypt a private message and create broadcast in one step
func (p *Packet) GetDecryptedPvtMess(sender *Client) *PrivateMessage {
	hlen := ((int(p.pay[0]) << 8) | int(p.pay[1])) - 2           // skip the sum
	nlen := ((int(p.pay[hlen+4]) << 8) | int(p.pay[hlen+5])) - 2 // skip the sum

	for i := 0; i < hlen; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	for i := 0; i < nlen; i++ {
		p.pay[hlen+8+i] = byte(p.pay[hlen+8+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}

	recipient := make([]byte, hlen)
	message := make([]byte, nlen)

	copy(recipient, p.pay[4:4+hlen])
	copy(message, p.pay[hlen+8:hlen+8+nlen])

	return NewPrivateMessage(sender.hnPair.handle, sender.hnPair.nickname, recipient, message)
}

func (p *Packet) GetCharacterStats() []byte {
	leng := int(0xD0)
	for i := 0; i < leng; i++ {
		p.pay[4+i] = byte(p.pay[4+i] ^ p.calcShift(byte(i), byte(p.pid&0xff)))
	}
	retval := make([]byte, leng)
	copy(retval, p.pay[4:4+leng])
	return retval
}

// decrypts and returns chat data
func (p *Packet) GetChatOutData() []byte {
	p.CryptString()

	leng := int(p.pay[0] << 8) | int(p.pay[1]) - 2
	retval := make([]byte, leng)
	copy(retval, p.pay[4:4+leng])
	return retval
}