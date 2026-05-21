package effect

type DamageSource string

const (
	DamageSourceValue      DamageSource = "value"
	DamageSourceUnitAttack DamageSource = "unit_attack"
)

type Damage struct {
	Value  int          `json:"value,omitzero"`
	Source DamageSource `json:"source"`
}

func (e *Damage) Apply() error {
	// TODO
	return nil
}
