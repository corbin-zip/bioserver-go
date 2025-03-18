package main

type Slots struct {
	slots []*Slot
	numberOfAreas int
	numberOfRooms int
	numberOfSlots int
}

// NewSlots creates a new Slots instance by allocating numberOfAreas * numberOfRooms * 20 slots.
func NewSlots(numberOfAreas, numberOfRooms int) *Slots {
    numberOfSlots := 20
    total := numberOfAreas * numberOfRooms * numberOfSlots
    slots := make([]*Slot, total)
    slotNum := 0
    for area := 0; area < numberOfAreas; area++ {
        for room := 0; room < numberOfRooms; room++ {
            for slot := 0; slot < numberOfSlots; slot++ {
                slots[slotNum] = NewSlot(area, room, slot)
                slotNum++
            }
        }
    }
    return &Slots{
        slots:         slots,
        numberOfAreas: numberOfAreas,
        numberOfRooms: numberOfRooms,
        numberOfSlots: numberOfSlots,
    }
}

// calcSlotnr calculates the index into the slots slice.
func (s *Slots) calcSlotnr(area, room, slotnr int) int {
    return slotnr + (room * s.numberOfSlots) + (area * s.numberOfRooms * s.numberOfSlots)
}

// GetSlot returns the Slot for the given area, room and slotnr.
func (s *Slots) GetSlot(area, room, slotnr int) *Slot {
    return s.slots[s.calcSlotnr(area, room, slotnr)]
}

// GetSlotCount returns the number of slots in a room.
func (s *Slots) GetSlotCount() int {
    return s.numberOfSlots
}

// GetStatus returns the status of a slot.
func (s *Slots) GetStatus(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetStatus()
}

// GetName returns the name (as a byte slice) of a slot.
func (s *Slots) GetName(area, room, slotnr int) []byte {
    return s.GetSlot(area, room, slotnr).GetName()
}

// GetScenario returns the scenario of a slot.
func (s *Slots) GetScenario(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetScenario()
}

// GetProtection returns the protection of a slot.
func (s *Slots) GetProtection(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetProtection()
}

// GetSlotType returns the slot type.
func (s *Slots) GetSlotType(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetSlotType()
}

// GetRulesCount returns the number of rules in the slot's ruleset.
func (s *Slots) GetRulesCount(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetRulesCount()
}

// GetRulesAttCount returns the number of attribute options for a rule.
func (s *Slots) GetRulesAttCount(area, room, slotnr, rulenr int) byte {
    return s.GetSlot(area, room, slotnr).GetRulesAttCount(rulenr)
}

// GetRuleName returns the name of a rule.
func (s *Slots) GetRuleName(area, room, slotnr, rulenr int) string {
    return s.GetSlot(area, room, slotnr).GetRuleName(rulenr)
}

// GetRuleValue returns the current value for a rule.
func (s *Slots) GetRuleValue(area, room, slotnr, rulenr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleValue(rulenr)
}

// GetRuleAttribute returns the attribute identifier for a rule.
func (s *Slots) GetRuleAttribute(area, room, slotnr, rulenr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleAttribute(rulenr)
}

// GetRuleAttributeDescription returns the description for an attribute option in a rule.
func (s *Slots) GetRuleAttributeDescription(area, room, slotnr, rulenr, attnr int) string {
    return s.GetSlot(area, room, slotnr).GetRuleAttributeDescription(rulenr, attnr)
}

// GetRuleAttributeAtt returns the attribute field for the given attribute option.
func (s *Slots) GetRuleAttributeAtt(area, room, slotnr, rulenr, attnr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleAttributeAtt(rulenr, attnr)
}

// GetDifficulty returns the difficulty level from the slot's ruleset.
func (s *Slots) GetDifficulty(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleSet().GetDifficulty()
}

// GetFriendlyFire returns the friendly fire value from the slot's ruleset.
func (s *Slots) GetFriendlyFire(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleSet().GetFriendlyFire()
}

// GetMaximumPlayers returns the maximum number of players from the slot's ruleset.
func (s *Slots) GetMaximumPlayers(area, room, slotnr int) byte {
    return s.GetSlot(area, room, slotnr).GetRuleSet().GetNumberOfPlayers()
}