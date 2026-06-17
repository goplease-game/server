// Package ability ...
package ability

import "github.com/goplease-game/server/ability/status"

// ID represents the unique identifier string for an ability.
type ID string

// Type defines the classification of the ability (e.g., Skill or Spell).
type Type int

// Ability type classifications.
const (
	Skill Type = iota + 1
	Spell
)

// ActivationType defines how an ability is targeted and triggered by the player.
type ActivationType string

// Supported activation and targeting mechanisms for abilities.
const (
	Instant          ActivationType = "instant"
	SelectAlly       ActivationType = "select_ally"
	SelectAllyOrSelf ActivationType = "select_ally_or_self"
	SelectEnemy      ActivationType = "select_enemy"
	SelectAnyUnit    ActivationType = "select_any_unit"
	SelectFreeCell   ActivationType = "select_free_cell"
	SelectAny        ActivationType = "select_any"
)

// AreaType defines the geometric shape of an ability's area of effect.
type AreaType string

// Supported area of effect shapes.
const (
	AreaLine   AreaType = "line"
	AreaCircle AreaType = "circle"
)

// TargetMode defines the filtering rules for valid units affected by an ability.
type TargetMode string

// Supported targeting filtration modes.
const (
	TargetSelf           TargetMode = "self"
	TargetAllies         TargetMode = "allies"
	TargetAlliesAndSelf  TargetMode = "allies_and_self"
	TargetEnemies        TargetMode = "enemies"
	TargetEnemiesAndSelf TargetMode = "enemies_and_self"
	TargetAny            TargetMode = "any"
)

// Ability defines the complete configuration, combat rules, and metadata of a game ability.
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

// Effect describes the mechanical changes applied to targets upon ability execution.
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

// ByID retrieves an Ability by its identifier from the global register.
// It populates and returns the Ability with the requested ID embedded.
func ByID(id ID) Ability {
	s, ok := abilities[id]
	if ok {
		s.ID = id
	}

	return s
}

// IsBasicAttack returns true if the ability corresponds to one of the default weapon attack actions.
func (a Ability) IsBasicAttack() bool {
	switch a.ID {
	case BasicMeleeAttack, BasicRangeAttack, BasicMagicAttack:
		return true
	}

	return false
}

// IsDirectDamage returns true if the ability deals targeted single-unit damage.
func (a Ability) IsDirectDamage() bool {
	if a.IsBasicAttack() {
		return true
	}

	if a.Activation == SelectEnemy && a.Effect.DealDamage != 0 {
		return true
	}

	return false
}
