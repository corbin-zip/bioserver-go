package main

type HNPairs struct {
	hnpairs [85]byte
	count   int
	length  int
}

func NewHNPairs() *HNPairs {
	return &HNPairs{
		count: 0,
		length: 1,
	}
}

func (hnp *HNPairs) Add(hnpair *HNPair) {
	hn := hnpair.GetHNPair()
	// Copy the HNPair into the buffer at index 'length'
	copy(hnp.hnpairs[hnp.length:], hn)
	hnp.length += len(hn)
	// Append two end marker bytes (0)
	hnp.hnpairs[hnp.length] = 0
	hnp.length++
	hnp.hnpairs[hnp.length] = 0
	hnp.length++
	hnp.count++
}

func (hnp *HNPairs) GetArray() []byte {
	// Set the count as the first byte
	hnp.hnpairs[0] = byte(hnp.count)
	// Return only the portion of the buffer that's been filled
	return hnp.hnpairs[:hnp.length]
}