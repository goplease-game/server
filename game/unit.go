package game

import (
	"errors"
	"fmt"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
	"github.com/ognev-dev/goplease/game/unit"
)

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

	Pos HexCoord `json:"pos"`

	Abilities []ability.ID                 `json:"abilities"`
	Cooldowns map[ability.ID]int           `json:"cooldowns"`
	Statuses  map[status.Type]status.Value `json:"statuses"`

	IsOpponent bool `json:"is_opponent"`
	IsDead     bool `json:"is_dead"`

	PhantomAPUsedThisTurn int `json:"phantom_ap_used_this_turn"`
}

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
		Pos:                   HexCoord{},
		Abilities:             t.Abilities,
		Cooldowns:             make(map[ability.ID]int),
		Statuses:              nil,
		IsOpponent:            false,
		IsDead:                false,
		PhantomAPUsedThisTurn: 0,
	}
}

func (u *Unit) ValidateAbilityUse(id ability.ID) error {
	if !u.HasAbility(id) {
		return fmt.Errorf("unit do not have ability: %s", string(id))
	}

	ab, ok := ability.Abilities[id]
	if !ok {
		return fmt.Errorf("unknown ability: %s", string(id))
	}

	if ab.Cooldown == 0 {
		return nil
	}

	cd, ok := u.Cooldowns[id]
	if !ok {
		return nil
	}

	if cd > 0 {
		return errors.New("ability is on cooldown")
	}

	return nil
}

func (u *Unit) HasAbility(id ability.ID) bool {
	for _, abID := range u.Abilities {
		if abID == id {
			return true
		}
	}

	return false
}

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

func (u *Unit) HasStatus(t status.Type) bool {
	_, ok := u.Statuses[t]
	return ok
}

func (u *Unit) AddStatus(value status.Value) {
	if u.Statuses == nil {
		u.Statuses = make(map[status.Type]status.Value)
	}

	u.Statuses[value.Status.Type] = value
}

func (u *Unit) RemoveStatus(t status.Type) {
	delete(u.Statuses, t)
}

func (u *Unit) IsEnemy(to *Unit) bool {
	return u.OwnerID != to.OwnerID
}

func (u *Unit) IsAlly(to *Unit) bool {
	return !u.IsEnemy(to)
}

func (u *Unit) Alive() bool {
	return !u.IsDead
}

func (u *Unit) AbilityReady(id ability.ID) bool {
	return !(u.Cooldowns[id] > 0)
}

func StartingUnits(playerID ds.ID) []*Unit {
	units := make([]*Unit, len(unit.DefaultTemplates))

	for i, t := range unit.DefaultTemplates {
		units[i] = NewUnitFromTemplate(t, playerID)
	}

	return units
}
