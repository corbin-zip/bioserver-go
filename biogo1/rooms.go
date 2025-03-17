package main

// TODO i hate this lol should probably replace it with a 2d array

const (
	NUMBER_OF_ROOMS int = 10
)

type Rooms struct {
	rooms         []*Room
	NumberOfAreas int
}

func NewRooms(numberOfAreas int) *Rooms {
	r := &Rooms{}
	r.rooms = make([]*Room, NUMBER_OF_ROOMS*numberOfAreas)
	curIndex := 0
	for i := 0; i < numberOfAreas; i++ {
		r.rooms[curIndex] = NewRoom(i, "R1", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R2", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R3", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R4", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R5", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R6", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R7", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R8", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "R9", STATUS_ACTIVE)
		curIndex++
		r.rooms[curIndex] = NewRoom(i, "RA", STATUS_ACTIVE)
		curIndex++
	}
	r.NumberOfAreas = numberOfAreas

	return r
}

func (r *Rooms) GetRoomCount() int {
	return len(r.rooms)
}

func (r *Rooms) GetName(areanr int, roomnr int) string {
	if areanr < 0 || areanr >= r.NumberOfAreas || roomnr < 0 || roomnr >= NUMBER_OF_ROOMS {
		return ""
	}
	return r.rooms[areanr*NUMBER_OF_ROOMS+roomnr-1].Name
}

func (r *Rooms) GetStatus(areanr int, roomnr int) byte {
	if areanr < 0 || areanr >= r.NumberOfAreas || roomnr < 0 || roomnr >= NUMBER_OF_ROOMS {
		return 0
	}
	return r.rooms[areanr*NUMBER_OF_ROOMS+roomnr-1].Status
}