package status

import "github.com/ognev-dev/goplease/app/ds"

type Type string
type Alignment string

const Permanent = -1

const (
	Provoked       Type = "provoked"
	Provoking      Type = "provoking"
	Stunned        Type = "stunned"
	Rallied        Type = "rallied"
	Marked         Type = "exposed"
	Hamstrung      Type = "hamstrung"
	Sharpened      Type = "sharpened"
	DebuffWard     Type = "debuff_ward"
	TemporalAnchor Type = "temporal_anchor"
	Frenzied       Type = "frenzied"
	Impatience     Type = "impatience"
)

const (
	Positive Alignment = "positive"
	Negative Alignment = "negative"
	Neutral  Alignment = "neutral"
)

type Status struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        Type   `json:"type"`
	// Duration -1 - permanent
	Duration     int       `json:"duration,omitempty"`
	InitialValue int       `json:"initial_value,omitempty"`
	Alignment    Alignment `json:"alignment"`
}

type Value struct {
	UnitID   ds.ID          `json:"unit_id"`
	Duration int            `json:"duration"`
	Value    int            `json:"value"`
	Status   *Status        `json:"status"`
	Meta     map[string]any `json:"meta"`
}

func (e Value) IsPositive() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Positive
}

func (e Value) IsNegative() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Negative
}

func (e Value) IsNeutral() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Neutral
}

var debuffWardStatus = &Status{
	Name:        "Debuff Ward",
	Description: "Prevents new debuffs from being applied for 1 turn.",
	Duration:    1,
	Type:        DebuffWard,
	Alignment:   Positive,
}

var sharpenedStatus = &Status{
	Name:         "Sharpened",
	Description:  "Increases attack by 1 until the end of the next turn.",
	Duration:     2,
	InitialValue: 1,
	Type:         Sharpened,
	Alignment:    Positive,
}

var hamstrungStatus = &Status{
	Name:        "Hamstrung",
	Description: "Took an arrow to the knee and cannot move.",
	Duration:    1,
	Type:        Hamstrung,
	Alignment:   Negative,
}

var markedStatus = &Status{
	Name:         "Marked",
	Description:  "Takes +1 damage from attacks. The unit that kills this unit permanently gains +1 Attack.",
	Duration:     Permanent,
	InitialValue: 1,
	Type:         Marked,
	Alignment:    Negative,
}

var stunnedStatus = &Status{
	Name:        "Stunned",
	Description: "This unit is stunned and cannot take its next action.",
	Duration:    1,
	Type:        Stunned,
	Alignment:   Negative,
}

var provokedStatus = &Status{
	Name:        "Provoked",
	Description: "Forced to target the provoker with the next direct-damage attack, if possible.",
	Duration:    1,
	Type:        Provoked,
	Alignment:   Negative,
}

// This status is only informational and has no effect
var provokingStatus = &Status{
	Name:        "Provoking",
	Description: "Forces nearby enemies to target you with their next direct-damage attack.",
	Duration:    1,
	Type:        Provoking,
	Alignment:   Neutral,
}

var ralliedStatus = &Status{
	Name:         "Rallied",
	Description:  "+1 Attack. Attacking an enemy makes this bonus permanent.",
	Duration:     1,
	InitialValue: 1,
	Type:         Rallied,
	Alignment:    Positive,
}

var temporalAnchorStatus = &Status{
	Name:         "Temporal Anchor",
	Description:  "Gain +1 AP at the start of your turn. At the end of the turn, restore your HP, Shield, and position to their state at the start of the turn.",
	InitialValue: 1,
	Duration:     1,
	Type:         TemporalAnchor,
	Alignment:    Positive,
}

var frenziedStatus = &Status{
	Name:         "Frenzied",
	Description:  "You are frenzied because 2 or more opponents are nearby. Grants +1 Attack while active.",
	InitialValue: 1,
	Type:         Frenzied,
	Alignment:    Positive,
	Duration:     Permanent,
}

var impatienceStatus = &Status{
	Name:         "Impatience",
	Description:  "Can't take this anymore. +1 Attack each turn until we're done here.",
	InitialValue: 1,
	Type:         Impatience,
	Alignment:    Positive,
	Duration:     Permanent,
}
