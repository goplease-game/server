package effect

type Effect struct {
	Damage *Damage `json:"damage,omitempty"`
	Heal   *Heal   `json:"heal,omitempty"`
	Status *Status `json:"status,omitempty"`
	Move   *Move   `json:"move,omitempty"`
	Dispel *Dispel `json:"dispel,omitempty"`
}

func NewDispelNegative() Effect {
	return Effect{
		Dispel: &Dispel{
			Level: DispelNegative,
		},
	}
}

func NewDispelPositive() Effect {
	return Effect{
		Dispel: &Dispel{
			Level: DispelPositive,
		},
	}
}

func NewAttack(value int) Effect {
	return Effect{
		Damage: &Damage{
			Source: DamageSourceValue,
			Value:  value,
		},
	}
}

func NewHeal(value int) Effect {
	return Effect{
		Heal: &Heal{
			Value: value,
		},
	}
}

func NewMove(t MoveType, dist int) Effect {
	return Effect{
		Move: &Move{
			Type:     t,
			Distance: dist,
		},
	}
}

func NewMoveTo() Effect {
	return Effect{
		Move: &Move{
			Type: MoveTo,
		},
	}
}

func NewMoveSwap() Effect {
	return Effect{
		Move: &Move{
			Type: MoveSwap,
		},
	}
}

func NewBasicAttack() Effect {
	return Effect{
		Damage: &Damage{
			Source: DamageSourceUnitAttack,
		},
	}
}

func NewStatusEffect(t StatusType) Effect {
	return Effect{
		Status: NewStatus(t),
	}
}
