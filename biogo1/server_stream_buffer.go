package main

import (
	"fmt"
)

const (
	RECEIVE_SIZE       = 8192 // Maximum buffer size.
	PACKET_HEADER_SIZE = 12   // Size of a packet header.
)

// ServerStreamBuffer manages incoming data using a fixed-size array.
// Instead of a dynamic slice, we use a [RECEIVE_SIZE]byte array and keep track
// of the number of valid bytes (buflen) and a pointer (messptr) to where the next
// complete message begins. This approach can help avoid reprocessing old data.
type ServerStreamBuffer struct {
	buf     [RECEIVE_SIZE]byte // Fixed-size byte array to hold incoming data.
	buflen  int                // The number of valid bytes currently stored in buf.
	messptr int                // Index into buf indicating the start of the next message.
}

// NewServerStreamBuffer creates a new ServerStreamBuffer with an empty buffer.
func NewServerStreamBuffer() *ServerStreamBuffer {
	return &ServerStreamBuffer{
		// buf is automatically zero-initialized.
		buflen:  0, // No data has been appended yet.
		messptr: 0, // Start processing at the beginning.
	}
}

// Append copies new data into the fixed buffer.
// It appends the incoming data at the current end of valid data (buflen)
// and then updates buflen. If the new data would overflow the buffer,
// it logs an error and ignores the extra data.
func (s *ServerStreamBuffer) Append(data []byte) []byte {
	n := len(data)
	if s.buflen+n > RECEIVE_SIZE {
		fmt.Println("Buffer overflow: cannot append data")
		return s.buf[:s.buflen] // Return what we have so far.
	}
	// Copy the incoming data into the buffer starting at index buflen.
	copy(s.buf[s.buflen:], data)
	s.buflen += n
	// Return a slice view of the valid portion of the buffer.
	return s.buf[:s.buflen]
}

// AppendData writes new incoming data into the buffer
func (s *ServerStreamBuffer) AppendData(data []byte) error {
	n := len(data)
	if s.buflen+n > RECEIVE_SIZE {
		return fmt.Errorf("buffer overflow")
	}
	copy(s.buf[s.buflen:], data)
	s.buflen += n
	return nil
}

// GetCompleteMessages scans the buffer for complete messages.
// It uses the header information (assumed to be PACKET_HEADERSIZE bytes)
// and a length field at offset 4-5 (big-endian) to determine message boundaries.
func (s *ServerStreamBuffer) GetCompleteMessages() []byte {
	size := s.buflen - s.messptr
	if size < PACKET_HEADER_SIZE {
		return nil
	}
	total := 0
	// Walk through the buffer from messptr, summing up complete messages.
	for {
		if size < PACKET_HEADER_SIZE {
			break
		}
		// Ensure that we have enough bytes for the header.
		if s.messptr+total+PACKET_HEADER_SIZE > s.buflen {
			break
		}
		// Calculate the payload length (plen) from the header bytes.
		// This mimics:
		// plen = (((int) b[messptr+total+4] << 8)&0xFF00) | ((int) b[messptr+total+5] &0xFF);
		plen := (int(s.buf[s.messptr+total+4]) << 8) | int(s.buf[s.messptr+total+5])
		// Check if we have the full message (header + payload)
		if size < plen+PACKET_HEADER_SIZE {
			break
		}
		total += plen + PACKET_HEADER_SIZE
		size = s.buflen - s.messptr - total
	}
	if total == 0 {
		return nil
	}
	// Extract the complete messages.
	retval := make([]byte, total)
	copy(retval, s.buf[s.messptr:s.messptr+total])
	// If no fragmented data remains, reset pointers; otherwise advance messptr.
	if size == 0 {
		s.buflen = 0
		s.messptr = 0
	} else {
		s.messptr += total
	}
	return retval
}

// // GetCompleteMessages scans the buffer for complete packets.
// // Each packet is assumed to have a fixed-size header (PACKET_HEADER_SIZE)
// // and a payload whose length is stored in the header at offsets 4 and 5.
// // This function extracts all complete packets, returns them concatenated,
// // and then shifts the buffer to remove the processed data.
// func (s *ServerStreamBuffer) GetCompleteMessages() []byte {
// 	// Calculate the number of bytes available for processing.
// 	size := s.buflen - s.messptr
// 	if size < PACKET_HEADER_SIZE {
// 		// Not enough data for even a header.
// 		return nil
// 	}
// 	ph.debug("Processing buffer, size: %d, messptr: %d\n", size, s.messptr)

// 	total := 0       // Total bytes that form complete messages.
// 	remaining := size // Bytes remaining to examine.

// 	// Loop through the data to find complete packets.
// 	for remaining > 0 {
// 		index := s.messptr + total
// 		// Ensure there's enough data for a full header.
// 		if index+PACKET_HEADER_SIZE > s.buflen {
// 			break
// 		}
// 		// Read the payload length from header bytes at offsets 4 and 5.
// 		plen := (int(s.buf[index+4]) << 8) | int(s.buf[index+5])
// 		// Check if the full packet (header + payload) is available.
// 		if remaining < (plen + PACKET_HEADER_SIZE) {
// 			break
// 		}
// 		// Accumulate the full packet size.
// 		total += plen + PACKET_HEADER_SIZE
// 		remaining -= plen + PACKET_HEADER_SIZE
// 	}
// 	if total == 0 {
// 		// No complete packets found.
// 		return nil
// 	}

// 	// Create a new slice to hold the complete messages.
// 	retval := make([]byte, total)
// 	copy(retval, s.buf[s.messptr:s.messptr+total])

// 	// Now, remove the processed data from the buffer.
// 	if remaining == 0 {
// 		// All data was processed; reset the buffer.
// 		fmt.Println("Resetting buffer since remaining is 0")
// 		s.messptr = 0
// 		s.buflen = 0
// 	} else {
// 		// There is leftover data that does not form a complete packet.
// 		// Shift the leftover data to the beginning of the buffer.
// 		ph.debug("Shifting buffer: remaining %d bytes\n", remaining)
// 		copy(s.buf[0:], s.buf[s.messptr+total:s.buflen])
// 		s.buflen = remaining
// 		s.messptr = 0
// 	}
// 	return retval
// }

// GetCompleteGameMessages processes game server packets that have a different format.
// If the first two bytes indicate a session packet (0x82, 0x02), it returns the entire buffer.
// Otherwise, it assumes each message starts with a one-byte length indicator.
func (s *ServerStreamBuffer) GetCompleteGameMessages() []byte {
	// Check if the buffer starts with the session packet marker.
	if s.buflen >= 2 && s.buf[0] == 0x82 && s.buf[1] == 0x02 {
		retval := make([]byte, s.buflen)
		copy(retval, s.buf[:s.buflen])
		s.buflen = 0
		s.messptr = 0
		return retval
	}
	size := s.buflen - s.messptr
	if size < 1 {
		return nil
	}
	total := 0
	remaining := size
	// Loop to process game messages, where the first byte is the message length.
	for remaining > 0 {
		index := s.messptr + total
		if index >= s.buflen {
			break
		}
		plen := int(s.buf[index]) & 0xFF
		if remaining < plen {
			break
		}
		total += plen
		remaining -= plen
	}
	if total == 0 {
		return nil
	}
	retval := make([]byte, total)
	copy(retval, s.buf[s.messptr:s.messptr+total])
	if remaining == 0 {
		s.messptr = 0
		s.buflen = 0
	} else {
		s.messptr += total
	}
	return retval
}

// func (s *ServerStreamBuffer) GetCompleteGameMessages() []byte {
// 	// Check if it is a session/gameplay packet.
// 	if s.buflen >= 2 && s.buf[0] == 0x82 && s.buf[1] == 0x02 {
// 		retval := make([]byte, s.buflen)
// 		copy(retval, s.buf[:s.buflen])
// 		s.buf = s.buf[:0]
// 		s.buflen = 0
// 		s.messptr = 0
// 		return retval
// 	}

// 	size := s.buflen - s.messptr
// 	if size < 1 {
// 		return nil
// 	}
// 	total := 0
// 	// Process lobby messages where each message’s first byte indicates its length.
// 	for size > 0 {
// 		if s.messptr+total >= s.buflen {
// 			break
// 		}
// 		plen := int(s.buf[s.messptr+total])
// 		size -= plen
// 		if size >= 0 {
// 			total += plen
// 		} else {
// 			break
// 		}
// 	}
// 	if total == 0 {
// 		return nil
// 	}
// 	retval := make([]byte, total)
// 	copy(retval, s.buf[s.messptr:s.messptr+total])
// 	if size == 0 {
// 		s.buf = s.buf[:0]
// 		s.buflen = 0
// 		s.messptr = 0
// 	} else {
// 		s.messptr += total
// 	}
// 	return retval
// }
