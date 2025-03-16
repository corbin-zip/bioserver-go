package main

type Area struct {
	nr          int
	name        string
	description string
	status      byte
}

func NewArea(number int, name string, description string, status byte) *Area {
	return &Area{
		nr: number,
		name: name,
		description: description,
		status: status,
	}
}