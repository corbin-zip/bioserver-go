package main

import (
	"fmt"
	"time"
)

const (
	STATUS_DISABLED byte = 0
	STATUS_FREE     byte = 1
	STATUS_INCREATE byte = 2
	STATUS_GAMESET  byte = 3
	STATUS_BUSY     byte = 4

	SCENARIO_TRAINING       byte = 0
	SCENARIO_WILDTHINGS     byte = 1
	SCENARIO_UNDERBELLY     byte = 2
	SCENARIO_FLASHBACK      byte = 3
	SCENARIO_DESPERATETIMES byte = 4
	SCENARIO_ENDOFTHEROAD   byte = 5
	SCENARIO_ELIMINATION1   byte = 6

	// TODO: there are more; what is 0x11?
	LOAD_NOTSET byte = 0
	LOAD_DVDROM byte = 1
	LOAD_HARDSK byte = 2

	PROTECTION_OFF byte = 0
	PROTECTION_ON  byte = 1

	WAITTIME_MILLSEC = 30 * 1000 * 1000
)

type Slot struct {
	area     int
	room     int
	slotnum  int
	gamenr   int
	betatest int

	name     []byte
	status   byte
	password []byte

	protection byte // using password?
	scenario   byte
	slottype   byte

	rules *RuleSet

	livetime int64

	host string //room master's userid
}

func NewSlot(area, room, slotnum int) *Slot {
	tmpname := fmt.Sprintf("a%d-r%d-s%d", area, room, slotnum)
	return &Slot{
		area:     area,
		room:     room,
		slotnum:  slotnum,
		gamenr:   0,
		betatest: 0,

		// name:     []byte("(free)"),
		name:       []byte(tmpname),
		status:     STATUS_FREE,
		scenario:   SCENARIO_TRAINING,
		slottype:   LOAD_NOTSET,
		protection: PROTECTION_OFF,
		rules:      NewRuleSet(),
		livetime:   -1,
	}
}

func (s *Slot) Reset() {
	s.gamenr = 0
	s.betatest = 0
	s.name = []byte("(free)")
	s.status = STATUS_FREE
	s.scenario = SCENARIO_TRAINING
	s.slottype = LOAD_NOTSET
	s.protection = PROTECTION_OFF
	s.rules.Reset()
	s.livetime = -1
}

// GetName returns the slot's name.
func (s *Slot) GetName() []byte {
	return s.name
}

// SetName sets the slot's name.
func (s *Slot) SetName(name []byte) {
	s.name = name
}

// GetPassword returns the slot's password.
func (s *Slot) GetPassword() []byte {
	return s.password
}

// SetPassword sets the slot's password and enables protection if non-empty.
func (s *Slot) SetPassword(passwd []byte) {
	s.password = passwd
	if len(passwd) > 0 {
		s.protection = PROTECTION_ON
	} else {
		s.protection = PROTECTION_OFF
	}
}

// GetStatus returns the slot's status.
func (s *Slot) GetStatus() byte {
	return s.status
}

// SetStatus sets the slot's status.
func (s *Slot) SetStatus(status byte) {
	s.status = status
}

// GetProtection returns the slot's protection status.
func (s *Slot) GetProtection() byte {
	return s.protection
}

// GetScenario returns the slot's scenario.
func (s *Slot) GetScenario() byte {
	return s.scenario
}

// SetScenario sets the slot's scenario.
func (s *Slot) SetScenario(scenario byte) {
	s.scenario = scenario
}

// GetSlotType returns the slot's type.
func (s *Slot) GetSlotType() byte {
	return s.slottype
}

// SetSlotType sets the slot's type.
func (s *Slot) SetSlotType(slottype byte) {
	s.slottype = slottype
}

// GetRulesCount returns the number of rules in the slot's ruleset.
func (s *Slot) GetRulesCount() byte {
	return byte(s.rules.GetRulesCount())
}

// GetRulesAttCount returns the number of attribute options for a given rule.
func (s *Slot) GetRulesAttCount(rulenr int) byte {
	return byte(s.rules.GetRulesAttCount(rulenr))
}

// GetRuleName returns the name for the given rule.
func (s *Slot) GetRuleName(rulenr int) string {
	return s.rules.GetRuleName(rulenr)
}

// GetRuleValue returns the value for the given rule.
func (s *Slot) GetRuleValue(rulenr int) byte {
	return s.rules.GetRuleValue(rulenr)
}

// SetRuleValue sets the value of the given rule.
func (s *Slot) SetRuleValue(rulenr int, value byte) {
	s.rules.SetRuleValue(rulenr, value)
}

// GetRuleAttribute returns the attribute for the given rule.
func (s *Slot) GetRuleAttribute(rulenr int) byte {
	return s.rules.GetRuleAttribute(rulenr)
}

// GetRuleAttributeDescription returns the description of the rule attribute option.
func (s *Slot) GetRuleAttributeDescription(rulenr, attnr int) string {
	return s.rules.GetRuleAttName(rulenr, attnr)
}

// GetRuleAttributeAtt returns the attribute field for the rule's attribute option.
func (s *Slot) GetRuleAttributeAtt(rulenr, attnr int) byte {
	return s.rules.GetRuleAttAttribute(rulenr, attnr)
}

// SetLivetime sets the slot's livetime to current time plus rules.WaitTime in minutes converted to milliseconds.
func (s *Slot) SetLivetime() {
	// Multiply wait time by 60*1000 to convert minutes to milliseconds.
	waitMillis := int64(s.rules.GetWaitTime()) * 60 * 1000
	s.livetime = time.Now().UnixMilli() + waitMillis
}

// GetLivetime returns the remaining livetime in seconds (0 if expired).
func (s *Slot) GetLivetime() int64 {
	remaining := (s.livetime - time.Now().UnixMilli()) / 1000
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRuleSet returns the slot's RuleSet.
func (s *Slot) GetRuleSet() *RuleSet {
	return s.rules
}

// SetHost sets the room master's userid.
func (s *Slot) SetHost(host string) {
	s.host = host
}

// GetHost returns the room master's userid.
func (s *Slot) GetHost() string {
	return s.host
}
