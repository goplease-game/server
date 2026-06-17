// Package status ...
package status

import "github.com/goplease-game/server/ds"

// Type identifies a kind of status effect that can be applied to a unit.
type Type string

// Alignment classifies whether a status is beneficial, harmful, or neutral.
type Alignment string

// Permanent is the Duration value indicating a status never expires.
const Permanent = -1

// Status type identifiers.
const (
	Provoked       Type = "provoked"
	Provoking      Type = "provoking"
	Stunned        Type = "stunned"
	Rallied        Type = "rallied"
	Marked         Type = "marked"
	Hamstrung      Type = "hamstrung"
	Sharpened      Type = "sharpened"
	DebuffWard     Type = "debuff_ward"
	TemporalAnchor Type = "temporal_anchor"
	Frenzied       Type = "frenzied"
	Impatience     Type = "impatience"
)

// Status alignment categories.
const (
	Positive Alignment = "positive"
	Negative Alignment = "negative"
	Neutral  Alignment = "neutral"
)

// Status describes the static definition of a status effect: its name,
// description, duration, initial value, and alignment.
type Status struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        Type   `json:"type"`
	// Duration -1 - permanent
	Duration     int       `json:"duration,omitempty"`
	InitialValue int       `json:"initial_value,omitempty"`
	Alignment    Alignment `json:"alignment"`
}

// Value represents an applied instance of a status on a specific unit,
// including its remaining duration, current value, and any per-application metadata.
type Value struct {
	UnitID   ds.ID          `json:"unit_id"`
	Duration int            `json:"duration"`
	Value    int            `json:"value"`
	Status   *Status        `json:"status"`
	Meta     map[string]any `json:"meta"`
}

// IsPositive reports whether the value's status has a positive alignment.
func (e Value) IsPositive() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Positive
}

// IsNegative reports whether the value's status has a negative alignment.
func (e Value) IsNegative() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Negative
}

// IsNeutral reports whether the value's status has a neutral alignment.
func (e Value) IsNeutral() bool {
	if e.Status == nil {
		return false
	}

	return e.Status.Alignment == Neutral
}

// debuffWardStatus is the static definition for the Debuff Ward status.
var debuffWardStatus = &Status{
	Name:        "Debuff Ward",
	Description: "Prevents new debuffs from being applied for 1 turn.",
	Duration:    1,
	Type:        DebuffWard,
	Alignment:   Positive,
}

// sharpenedStatus is the static definition for the Sharpened status.
var sharpenedStatus = &Status{
	Name:         "Sharpened",
	Description:  "Increases attack by 1 until the end of the next turn.",
	Duration:     2,
	InitialValue: 1,
	Type:         Sharpened,
	Alignment:    Positive,
}

// hamstrungStatus is the static definition for the Hamstrung status.
var hamstrungStatus = &Status{
	Name:        "Hamstrung",
	Description: "Took an arrow to the knee and cannot move.",
	Duration:    1,
	Type:        Hamstrung,
	Alignment:   Negative,
}

// markedStatus is the static definition for the Marked status.
var markedStatus = &Status{
	Name:         "Marked",
	Description:  "Takes +1 damage from attacks. The unit that kills this unit permanently gains +1 Attack.",
	Duration:     Permanent,
	InitialValue: 1,
	Type:         Marked,
	Alignment:    Negative,
}

// stunnedStatus is the static definition for the Stunned status.
var stunnedStatus = &Status{
	Name:        "Stunned",
	Description: "This unit is stunned and cannot take its next action.",
	Duration:    1,
	Type:        Stunned,
	Alignment:   Negative,
}

// provokedStatus is the static definition for the Provoked status.
var provokedStatus = &Status{
	Name:        "Provoked",
	Description: "Forced to target the provoker with the next direct-damage attack, if possible.",
	Duration:    1,
	Type:        Provoked,
	Alignment:   Negative,
}

// provokingStatus is the static definition for the Provoking status.
// This status is only informational and has no effect.
var provokingStatus = &Status{
	Name:        "Provoking",
	Description: "Forces nearby enemies to target you with their next direct-damage attack.",
	Duration:    1,
	Type:        Provoking,
	Alignment:   Neutral,
}

// ralliedStatus is the static definition for the Rallied status.
var ralliedStatus = &Status{
	Name:         "Rallied",
	Description:  "+1 Attack. Attacking an enemy makes this bonus permanent.",
	Duration:     1,
	InitialValue: 1,
	Type:         Rallied,
	Alignment:    Positive,
}

// temporalAnchorStatus is the static definition for the Temporal Anchor status.
var temporalAnchorStatus = &Status{
	Name:         "Temporal Anchor",
	Description:  "Gain +1 AP at the start of your turn. At the end of the turn, restore your HP, Shield, and position to their state at the start of the turn.",
	InitialValue: 1,
	Duration:     1,
	Type:         TemporalAnchor,
	Alignment:    Positive,
}

// frenziedStatus is the static definition for the Frenzied status.
var frenziedStatus = &Status{
	Name:         "Frenzied",
	Description:  "You are frenzied because 2 or more opponents are nearby. Grants +1 Attack while active.",
	InitialValue: 1,
	Type:         Frenzied,
	Alignment:    Positive,
	Duration:     Permanent,
}

// impatienceStatus is the static definition for the Impatience status.
var impatienceStatus = &Status{
	Name:         "Impatience",
	Description:  "Can't take this anymore. +1 Attack each turn until we're done here.",
	InitialValue: 1,
	Type:         Impatience,
	Alignment:    Positive,
	Duration:     Permanent,
}
