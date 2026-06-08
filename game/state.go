package game

import (
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

// ApplyState represents a single, atomic state mutation applied to a unit or player.
// Sequential execution of these states forms the visual timeline on the client side.
type ApplyState struct {
	ToUnitID ds.ID `json:"to_unit_id,omitempty"`

	SetPhantomAP *int `json:"set_phantom_ap,omitempty"`

	SkipTurn bool    `json:"skip_turn,omitempty"`
	ShowText *string `json:"show_text,omitempty"`

	// Movement
	MoveTo *HexCoord `json:"move_to,omitempty"` // New position on the grid

	// Delta changes used to trigger floating text or combat UI animations
	ChangeHP     *int `json:"change_hp,omitempty"`
	ChangeAP     *int `json:"change_ap,omitempty"`
	ChangeMP     *int `json:"change_mp,omitempty"`
	ChangeShield *int `json:"change_shield,omitempty"`
	ChangeAtk    *int `json:"change_atk,omitempty"`

	// Absolute values used for hard state synchronization after the animation plays
	SetHP     *int `json:"set_hp,omitempty"`
	SetBaseHP *int `json:"set_base_hp,omitempty"`
	SetAP     *int `json:"set_ap,omitempty"`
	SetMP     *int `json:"set_mp,omitempty"`
	SetShield *int `json:"set_shield,omitempty"`
	SetAtk    *int `json:"set_atk,omitempty"`

	SetCooldown *map[ability.ID]int `json:"set_cooldown,omitempty"`

	// Statuses and effects
	IsDead        bool           `json:"is_dead,omitempty"`
	AddStatus     *status.Type   `json:"add_status,omitempty"`
	AddStatusMeta map[string]any `json:"add_status_meta,omitempty"`
	RemoveStatus  *status.Type   `json:"remove_status,omitempty"`

	SetStatusDuration map[status.Type]int `json:"set_status_duration,omitempty"`

	UseAbility *UseAbilityPayload `json:"use_ability,omitempty"`
}

// ApplyStates represents a collection of atomic state mutations bound to a single unit.
type ApplyStates []ApplyState

// NewUnitStates initializes a new ApplyStates slice dedicated to a specific unit's timeline events.
func NewUnitStates(ss ...ApplyState) ApplyStates {
	return ss
}

// Add appends new states to this unit's timeline.
func (s *ApplyStates) Add(ss ...ApplyState) ApplyStates {
	*s = append(*s, ss...)
	return *s
}

type UseAbilityPayload struct {
	UnitID    string     `json:"unit_id"`
	AbilityID ability.ID `json:"ability_id"`
	Target    HexCoord   `json:"target,omitempty"`
}
