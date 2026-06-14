package status

var statuses = map[Type]*Status{
	Rallied:        ralliedStatus,
	Provoked:       provokedStatus,
	Provoking:      provokingStatus,
	Stunned:        stunnedStatus,
	Hamstrung:      hamstrungStatus,
	Marked:         markedStatus,
	Sharpened:      sharpenedStatus,
	DebuffWard:     debuffWardStatus,
	TemporalAnchor: temporalAnchorStatus,
	Frenzied:       frenziedStatus,
	Impatience:     impatienceStatus,
}

// ByType returns the Status definition for the given Type, or nil if not found.
func ByType(t Type) *Status {
	return statuses[t]
}
