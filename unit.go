package game

import (
	"errors"
	"fmt"
	"slices"

	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
	"github.com/goplease-game/server/unit"
)

var (
	// ErrAbilityOnCooldown indicates that the ability is currently on cooldown and cannot be used.
	ErrAbilityOnCooldown = errors.New("on cooldown")

	// ErrUnitDoNotHaveAbility indicates that the unit does not possess the requested ability.
	ErrUnitDoNotHaveAbility = errors.New("unit do not have ability")

	// ErrUnknownAbility indicates that the requested ability cannot be found or is not registered.
	ErrUnknownAbility = errors.New("unknown ability")
)

// Unit represents a single combat unit's authoritative server-side state,
// including its stats, abilities, cooldowns, and statuses.
type Unit struct {
	ID          ds.ID  `json:"id"`
	TemplateID  int    `json:"template_id"`
	OwnerID     ds.ID  `json:"owner_id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	BaseAtk       int `json:"base_atk"`
	CurrentAtk    int `json:"current_atk"`
	BaseHP        int `json:"base_hp"`
	CurrentHP     int `json:"current_hp"`
	CurrentShield int `json:"current_shield"`
	BaseAP        int `json:"base_ap"` // Action Points
	CurrentAP     int `json:"current_ap"`
	BaseMP        int `json:"base_mp"` // Move Points
	CurrentMP     int `json:"current_mp"`

	Pos *HexCoord `json:"pos"`

	Abilities []ability.ID                 `json:"abilities"`
	Cooldowns map[ability.ID]int           `json:"cooldowns"`
	Statuses  map[status.Type]status.Value `json:"statuses"`

	IsOpponent bool `json:"is_opponent"`
	IsDead     bool `json:"is_dead"`

	PhantomAPUsedThisTurn int `json:"phantom_ap_used_this_turn"`
}

// NewUnitFromTemplate creates a new Unit for playerID from the given
// template, with current stats initialized to their base values and no
// position assigned.
func NewUnitFromTemplate(t unit.Template, playerID ds.ID) *Unit {
	return &Unit{
		ID:                    ds.NewID(),
		TemplateID:            t.ID,
		OwnerID:               playerID,
		Name:                  t.Name,
		Description:           t.Description,
		BaseAtk:               t.Attack,
		CurrentAtk:            t.Attack,
		BaseHP:                t.HP,
		CurrentHP:             t.HP,
		CurrentShield:         t.Shield,
		BaseAP:                t.ActionPoints,
		CurrentAP:             t.ActionPoints,
		BaseMP:                t.MovePoints,
		CurrentMP:             t.MovePoints,
		Pos:                   nil,
		Abilities:             t.Abilities,
		Cooldowns:             make(map[ability.ID]int),
		Statuses:              nil,
		IsOpponent:            false,
		IsDead:                false,
		PhantomAPUsedThisTurn: 0,
	}
}

// PosVal returns the unit's position as a value type.
// Panics if the unit has not been placed on the board yet.
// Use instead of dereferencing Pos directly in handlers where
// the unit is guaranteed to be on the board.
func (u *Unit) PosVal() HexCoord {
	if u.Pos == nil {
		panic(fmt.Sprintf("unit %s has no position", u.ID))
	}
	return *u.Pos
}

// ValidateAbilityUse reports an error if the unit cannot currently use the
// given ability: if it doesn't have the ability, the ability is unknown,
// or the ability is on cooldown.
func (u *Unit) ValidateAbilityUse(id ability.ID) error {
	if !u.HasAbility(id) {
		return fmt.Errorf("%w: %s", ErrUnitDoNotHaveAbility, string(id))
	}

	ab := ability.ByID(id)
	if ab.ID == ability.Unknown {
		return fmt.Errorf("%w: %s", ErrUnknownAbility, string(id))
	}

	if ab.Cooldown == 0 {
		return nil
	}

	if !u.AbilityReady(id) {
		return fmt.Errorf("ability %s: %w", ab.ID, ErrAbilityOnCooldown)
	}

	return nil
}

// HasAbility reports whether the unit has the given ability.
func (u *Unit) HasAbility(id ability.ID) bool {
	return slices.Contains(u.Abilities, id)
}

// SetCooldown sets the cooldown for the given ability, removing the entry
// entirely when cd is zero.
func (u *Unit) SetCooldown(id ability.ID, cd int) {
	if u.Cooldowns == nil {
		u.Cooldowns = make(map[ability.ID]int)
	}

	if cd == 0 {
		delete(u.Cooldowns, id)
		return
	}

	u.Cooldowns[id] = cd
}

// HasStatus reports whether the unit currently has the given status type.
func (u *Unit) HasStatus(t status.Type) bool {
	_, ok := u.Statuses[t]
	return ok
}

// AddStatus applies the given status value to the unit, replacing any
// existing value for the same status type.
func (u *Unit) AddStatus(value status.Value) {
	if u.Statuses == nil {
		u.Statuses = make(map[status.Type]status.Value)
	}

	u.Statuses[value.Status.Type] = value
}

// RemoveStatus removes the given status type from the unit.
func (u *Unit) RemoveStatus(t status.Type) {
	delete(u.Statuses, t)
}

// IsEnemy reports whether the given unit belongs to a different owner.
func (u *Unit) IsEnemy(to *Unit) bool {
	return u.OwnerID != to.OwnerID
}

// IsAlly reports whether the given unit belongs to the same owner.
func (u *Unit) IsAlly(to *Unit) bool {
	return !u.IsEnemy(to)
}

// Alive reports whether the unit is still alive.
func (u *Unit) Alive() bool {
	return !u.IsDead
}

// AbilityReady reports whether the given ability is off cooldown.
func (u *Unit) AbilityReady(id ability.ID) bool {
	return u.Cooldowns[id] <= 0
}

// StartingUnits creates a unit for each default template, all owned by playerID.
func StartingUnits(playerID ds.ID) []*Unit {
	units := make([]*Unit, len(unit.DefaultTemplates))

	for i, t := range unit.DefaultTemplates {
		units[i] = NewUnitFromTemplate(t, playerID)
	}

	return units
}
