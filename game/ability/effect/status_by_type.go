package effect

var statusByType = map[StatusType]*Status{
	DecayingShield: decayingShieldStatus,
	DecayingAttack: decayingAttackStatus,
	Provoked:       provokedStatus,
	Stun:           stunStatus,
	Hamstrung:      hamstrungStatus,
	Exposed:        exposedStatus,
	Sharpened:      sharpenedStatus,
	DebuffWard:     debuffWardStatus,
}

func NewStatus(t StatusType) *Status {
	return statusByType[t]
}
