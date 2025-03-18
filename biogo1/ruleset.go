package main

type Rule struct {
	name      string
	attribute byte // todo: what happens if <> 1?
	value     byte
}

func NewRule(name string, attribute byte, value byte) *Rule {
	return &Rule{
		name:      name,
		attribute: (attribute & 0xff),
		value:     (value & 0xff),
	}
}

// RuleSet contains the standard ruleset and associated attribute options.
type RuleSet struct {
	ruleset    []*Rule
	attributes [][]*Rule
}

// NewRuleSet creates a new RuleSet with the standard settings.
// Standard settings:
//
//	ruleset[0]: "number of players", attribute 1, value 2
//	ruleset[1]: "wait limit",        attribute 1, value 2
//	ruleset[2]: "difficulty level",  attribute 1, value 3
//	ruleset[3]: "friendly fire",     attribute 1, value 0
//
// And the associated attributes arrays as defined in the Java version.
func NewRuleSet() *RuleSet {
	rs := &RuleSet{
		ruleset: []*Rule{
			NewRule("number of players", 1, 2),
			NewRule("wait limit", 1, 2),
			NewRule("difficulty level", 1, 3),
			NewRule("friendly fire", 1, 0),
		},
		attributes: [][]*Rule{
			{
				NewRule("two players", 0, 0),
				NewRule("three players", 0, 0),
				NewRule("four players", 0, 0),
			},
			{
				NewRule("three minutes", 0, 0),
				NewRule("five minutes", 0, 0),
				NewRule("ten minutes", 0, 0),
				NewRule("fifteen minutes", 0, 0),
				NewRule("thirty minutes", 0, 0),
			},
			{
				NewRule("easy", 0, 0),
				NewRule("normal", 0, 0),
				NewRule("hard", 0, 0),
				NewRule("very hard", 0, 0),
			},
			{
				NewRule("off", 0, 0),
				NewRule("on", 0, 0),
			},
		},
	}
	return rs
}

// GetRuleField is a helper function to return the database field for a given rule number.
func GetRuleField(area int, rulenr byte) string {
	switch rulenr {
	case 0:
		return "maxplayers"
	case 1:
		// In Java this returned null. In Go we return an empty string.
		return ""
	case 2:
		return "difficulty"
	case 3:
		return "friendlyfire"
	default:
		return ""
	}
}

// Reset resets the ruleset values to the standard settings.
func (rs *RuleSet) Reset() {
	rs.ruleset[0].value = 2
	rs.ruleset[1].value = 2
	rs.ruleset[2].value = 3
	rs.ruleset[3].value = 0
}

// GetRuleName returns the name of the rule at the given index.
func (rs *RuleSet) GetRuleName(nr int) string {
	return rs.ruleset[nr].name
}

// GetRuleAttribute returns the attribute of the rule at the given index.
func (rs *RuleSet) GetRuleAttribute(nr int) byte {
	return rs.ruleset[nr].attribute
}

// GetRuleAttName returns the name for the attribute option in rule nr at attribute index nratt.
func (rs *RuleSet) GetRuleAttName(nr int, nratt int) string {
	return rs.attributes[nr][nratt].name
}

// GetRuleAttAttribute returns the attribute field for the attribute option in rule nr at index nratt.
func (rs *RuleSet) GetRuleAttAttribute(nr int, nratt int) byte {
	return rs.attributes[nr][nratt].attribute
}

// GetRulesCount returns the number of rules in the ruleset.
func (rs *RuleSet) GetRulesCount() int {
	return len(rs.ruleset)
}

// GetRulesAttCount returns the number of attribute options for the rule at the given index.
func (rs *RuleSet) GetRulesAttCount(rulenr int) int {
	return len(rs.attributes[rulenr])
}

// GetRuleValue returns the current value for the rule at the given index.
func (rs *RuleSet) GetRuleValue(rulenr int) byte {
	return rs.ruleset[rulenr].value
}

// SetRuleValue sets the value for the rule at the given index.
func (rs *RuleSet) SetRuleValue(rulenr int, value byte) {
	rs.ruleset[rulenr].value = value
}

// GetDifficulty returns the difficulty level value.
func (rs *RuleSet) GetDifficulty() byte {
	return rs.ruleset[2].value
}

// GetFriendlyFire returns the friendly fire value.
func (rs *RuleSet) GetFriendlyFire() byte {
	return rs.ruleset[3].value
}

// GetWaitTime returns the wait time in minutes corresponding to the wait limit rule.
// Mapping based on rule value:
//
//	0 -> 3, 1 -> 5, 2 -> 10, 3 -> 15, 4 -> 30, default -> 30
func (rs *RuleSet) GetWaitTime() int64 {
	switch rs.ruleset[1].value {
	case 0:
		return 3
	case 1:
		return 5
	case 2:
		return 10
	case 3:
		return 15
	case 4:
		return 30
	default:
		return 30
	}
}

// GetNumberOfPlayers returns the number of players based on the rule value.
// Mapping based on rule value:
//
//	0 -> 2, 1 -> 3, 2 -> 4, default -> 2
func (rs *RuleSet) GetNumberOfPlayers() byte {
	switch rs.ruleset[0].value {
	case 0:
		return 2
	case 1:
		return 3
	case 2:
		return 4
	default:
		return 2
	}
}
