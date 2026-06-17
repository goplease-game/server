package game

import (
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
)

// ApplyState represents a single, atomic state mutation applied to a unit or player.
// Sequential execution of these states forms the visual timeline on the client side.
type ApplyState struct {
	ToUnitID ds.ID `json:"to_unit_id,omitzero"`

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

// ApplyStates collects the ApplyState mutations produced by a single game
// action, split by audience: Global (broadcast to both players), Self
// (sent only to the acting player), and Opponent (sent only to the
// opponent).
type ApplyStates struct {
	Global   []ApplyState
	Self     []ApplyState
	Opponent []ApplyState
}

// ToAll appends the given states to Global, to be broadcast to both players.
func (s *ApplyStates) ToAll(ss ...ApplyState) {
	s.Global = append(s.Global, ss...)
}

// ToSelf appends the given states to Self, to be sent only to the acting player.
func (s *ApplyStates) ToSelf(ss ...ApplyState) {
	s.Self = append(s.Self, ss...)
}

// ToOpp appends the given states to Opponent, to be sent only to the opponent.
func (s *ApplyStates) ToOpp(ss ...ApplyState) {
	s.Opponent = append(s.Opponent, ss...)
}

// With merges another ApplyStates into s, appending each of its Global,
// Self, and Opponent slices.
func (s *ApplyStates) With(state ApplyStates) {
	s.Global = append(s.Global, state.Global...)
	s.Self = append(s.Self, state.Self...)
	s.Opponent = append(s.Opponent, state.Opponent...)
}

// IsEmpty reports whether there are no states to apply.
func (s *ApplyStates) IsEmpty() bool {
	return len(s.Global) == 0 && len(s.Opponent) == 0 && len(s.Self) == 0
}

// HasSkipTurn reports whether any of the Self or Global states have SkipTurn set.
func (s *ApplyStates) HasSkipTurn() bool {
	for _, st := range s.Self {
		if st.SkipTurn {
			return true
		}
	}
	for _, st := range s.Global {
		if st.SkipTurn {
			return true
		}
	}

	return false
}
