package effect

type StatusType string
type StatusAlignment string

const (
	DecayingShield StatusType = "decaying_shield"
	Provoked       StatusType = "provoked"
	Stun           StatusType = "stun"
	DecayingAttack StatusType = "decaying_attack"
	Exposed        StatusType = "exposed"
	Hamstrung      StatusType = "hamstrung"
	Sharpened      StatusType = "sharpened"
	DebuffWard     StatusType = "debuff_ward"
)

const (
	Positive StatusAlignment = "positive"
	Negative StatusAlignment = "negative"
	Neutral  StatusAlignment = "neutral"
)

type Status struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Type         StatusType      `json:"type"`
	Duration     int             `json:"duration,omitempty"`
	InitialValue int             `json:"initial_value,omitempty"`
	Alignment    StatusAlignment `json:"alignment"`
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
	Description:  "Increases attack by 2 until the end of the next turn.",
	Duration:     1,
	InitialValue: 2,
	Type:         Sharpened,
	Alignment:    Positive,
}

var hamstrungStatus = &Status{
	Name:        "Hamstrung",
	Description: "Movement is reduced to 1 tile per turn.",
	Duration:    1,
	Type:        Hamstrung,
	Alignment:   Negative,
}

var exposedStatus = &Status{
	Name:        "Exposed",
	Description: "Attacks against this unit deal +1 damage.",
	Duration:    3,
	Type:        Exposed,
	Alignment:   Negative,
}

var stunStatus = &Status{
	Name:        "Stunned",
	Description: "This unit is stunned and cannot take its next action.",
	Duration:    1,
	Type:        Stun,
	Alignment:   Negative,
}

var provokedStatus = &Status{
	Name:        "Provoked",
	Description: "This unit is forced to target the provoking unit if possible.",
	Duration:    1,
	Type:        Provoked,
	Alignment:   Negative,
}

var decayingAttackStatus = &Status{
	Name:         "Rallied",
	Description:  "A bonus that increases your attack. It decays by 1 at the end of each turn.",
	Duration:     3,
	InitialValue: 3,
	Type:         DecayingAttack,
	Alignment:    Positive,
}

var decayingShieldStatus = &Status{
	Name:         "Shield",
	Description:  "A shield that protect your health. It decays by 1 at the end of each turn.",
	Duration:     3,
	InitialValue: 3,
	Type:         DecayingShield,
	Alignment:    Positive,
}
