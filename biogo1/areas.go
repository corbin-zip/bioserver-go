package main

const (
	STATUS_ACTIVE   byte = 3
	STATUS_INACTIVE byte = 0
)

type Areas struct {
	areas []*Area
}

func NewAreas() *Areas {
	a := &Areas{}
	a.areas = append(a.areas,
		NewArea(1, "East Town", "<BODY><SIZE=3>standard rules<END>", STATUS_ACTIVE),
		NewArea(2, "West Town", "<BODY><SIZE=3>individual games<END>", STATUS_ACTIVE),
	)
	return a
}

func (a *Areas) GetAreaCount() int {
	return len(a.areas)
}

func (a *Areas) GetName(areaNumber int) string {
	// Java code uses areas.get(areanumber-1)
	if areaNumber <= 0 || areaNumber > len(a.areas) {
		return ""
	}
	return a.areas[areaNumber-1].name
}

func (a *Areas) GetDescription(areaNumber int) string {
	if areaNumber <= 0 || areaNumber > len(a.areas) {
		return ""
	}
	return a.areas[areaNumber-1].description
}

func (a *Areas) GetStatus(areaNumber int) byte {
	if areaNumber <= 0 || areaNumber > len(a.areas) {
		return 0
	}
	return a.areas[areaNumber-1].status
}