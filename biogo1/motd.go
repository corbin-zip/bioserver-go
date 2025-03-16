package main

type MOTD struct {
	number  byte
	message string
}

func NewMOTD(number int, message string) *MOTD {
	return &MOTD{
		number:  byte(number),
		message: message,
	}
}

func (m *MOTD) GetPacket() []byte {
	mlen := len(m.message)
	retval := make([]byte, mlen+3)
	retval[0] = m.number
	if mlen == 0 {
		retval[0] = 0
	}
	retval[1] = byte(mlen >> 8)
	retval[2] = byte(mlen & 0xFF)
	copy(retval[3:], []byte(m.message))
	return retval
}
