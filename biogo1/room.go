package main

type Room struct {
	Name string
	Status byte
	AreaNumber int
}

func NewRoom(area int, name string, status byte) *Room {
	return &Room{
		Name: name,
		Status: status,
		AreaNumber: area,
	}
}