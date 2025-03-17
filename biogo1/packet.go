package main

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
	return int(uint16((p.pay[0])<<8) | uint16(p.pay[1]))
}
