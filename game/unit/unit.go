package unit

import (
	"errors"
	"fmt"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/effect"
)

type Template struct {
	ID           int
	Name         string
	Description  string
	HP           int
	Attack       int
	AttackRange  int
	Shield       int
	MovePoints   int
	ActionPoints int
	Abilities    []ability.ID
}

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

	BaseAP    int `json:"base_ap"` // Action Points
	CurrentAP int `json:"current_ap"`
	MP        int `json:"mp"` // Move Points

	// board position, -1 - in hand
	Row int `json:"row"`
	Col int `json:"col"`

	Abilities []ability.ID        `json:"abilities"`
	Cooldowns map[ability.ID]int  `json:"cooldowns"`
	Statuses  []effect.UnitStatus `json:"statuses"`
}

func NewUnitFromTemplate(t Template, playerID ds.ID) *Unit {
	return &Unit{
		ID:            ds.NewID(),
		TemplateID:    t.ID,
		OwnerID:       playerID,
		Name:          t.Name,
		Description:   t.Description,
		BaseAtk:       t.Attack,
		CurrentAtk:    t.Attack,
		BaseHP:        t.HP,
		CurrentHP:     t.HP,
		CurrentShield: t.Shield,
		BaseAP:        t.ActionPoints,
		CurrentAP:     t.ActionPoints,
		MP:            t.MovePoints,
		Row:           -1,
		Col:           -1,
		Abilities:     t.Abilities,
		Cooldowns:     make(map[ability.ID]int),
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

func (u *Unit) IsAlive() bool { return u.CurrentHP > 0 }
func (u *Unit) InHand() bool  { return u.Row == -1 }
