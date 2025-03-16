package main

import (
	"io"
	"log"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

type HNPair struct {
	handle   []byte
	nickname []byte
}

func NewHNPairFromBytes(handle, nickname []byte) *HNPair {
	return &HNPair{handle, nickname}
}

func NewHNPairFromStrings(handle, nickname string) *HNPair {
	handleBytes := []byte(handle)
	encoder := japanese.ShiftJIS.NewEncoder()
	reader := transform.NewReader(strings.NewReader(nickname), encoder)
	nickBytes, err := io.ReadAll(reader)
	if err != nil {
		// Fallback to "sjis" if encoding fails
		nickBytes = []byte("sjis")
		log.Printf("ShiftJIS encoding failed: %v", err)
	}
	return &HNPair{handleBytes, nickBytes}
}

func (hnp *HNPair) GetHNPair() []byte {
	hnpair := make([]byte, len(hnp.handle)+len(hnp.nickname)+4)
	hnpair[0] = 0
	hnpair[1] = 6
	copy(hnpair[2:8], hnp.handle) // assuming handle is exactly 6 bytes
	hnpair[8] = 0
	hnpair[9] = byte(len(hnp.nickname))
	copy(hnpair[10:], hnp.nickname)
	return hnpair
}
