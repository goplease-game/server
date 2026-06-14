package ability

import "github.com/ognev-dev/goplease/game/ability/status"

type ID string

type Type int

const (
	Skill Type = iota + 1
	Spell
)

type ActivationType string

const (
	Instant          ActivationType = "instant"
	SelectAlly       ActivationType = "select_ally"
	SelectAllyOrSelf ActivationType = "select_ally_or_self"
	SelectEnemy      ActivationType = "select_enemy"
	SelectAnyUnit    ActivationType = "select_any_unit"
	SelectFreeCell   ActivationType = "select_free_cell"
	SelectAny        ActivationType = "select_any"
)

type AreaType string

const (
	AreaLine   AreaType = "line"
	AreaCircle AreaType = "circle"
)

type TargetMode string

const (
	TargetSelf           TargetMode = "self"
	TargetAllies         TargetMode = "allies"
	TargetAlliesAndSelf  TargetMode = "allies_and_self"
	TargetEnemies        TargetMode = "enemies"
	TargetEnemiesAndSelf TargetMode = "enemies_and_self"
	TargetAny            TargetMode = "any"
)

type Ability struct {
	ID          ID     `json:"id"`
	Type        Type   `json:"type"`
	IsPassive   bool   `json:"is_passive"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cooldown    int    `json:"cooldown"`
	DamageHint  string `json:"damage_hint"`

	Range      int            `json:"range"`
	TargetMode TargetMode     `json:"target_mode"`
	Activation ActivationType `json:"activation"`
	Area       AreaType       `json:"area"`
	AreaRadius int            `json:"area_radius"`

	Effect Effect `json:"effect"`
}

type Effect struct {
	HealHP        int         `json:"heal_hp"`
	AddHP         int         `json:"add_hp"`
	AddShield     int         `json:"add_shield"`
	AddAP         int         `json:"add_ap"`
	AddAtk        int         `json:"add_atk"`
	DealDamage    int         `json:"deal_damage"`
	DealAltDamage int         `json:"deal_alt_damage"`
	BonusDamage   int         `json:"bonus_damage"`
	ApplyStatus   status.Type `json:"apply_status"`
}

func ByID(id ID) Ability {
	s, ok := Abilities[id]
	if ok {
		s.ID = id
	}

	return s
}

func (a Ability) IsBasicAttack() bool {
	switch a.ID {
	case BasicMeleeAttack, BasicRangeAttack, BasicMagicAttack:
		return true
	}

	return false
}
