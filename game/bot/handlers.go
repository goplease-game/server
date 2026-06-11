package bot

import (
	"encoding/json"
	"log"

	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/api"
)

func (b *Bot) handleNewGame(data json.RawMessage) {
	var ng game.NewGamePayload
	err := json.Unmarshal(data, &ng)
	if err != nil {
		log.Printf("[bot] [new_game] unmarshal error: %s", err)
	}

	log.Printf("[bot] starting new game with %s", ng.Opponent)

	b.state = &gameState{
		board:  ng.Board,
		player: ng.Player,
	}

	b.reply(api.ReadyToPlay, nil)
}

func (b *Bot) handleOpponentUnitPlaced(data json.RawMessage) {
	var load game.PlaceUnitPayload
	if err := json.Unmarshal(data, &load); err != nil {
		log.Printf("[bot] [unit_placed] unmarshal error: %s", err)
		return
	}

	load.Unit.Pos = &load.Coord

	cell := b.state.board.Cells[load.Coord]
	if cell == nil {
		log.Printf("[bot] [unit_placed] cell not found at %s", load.Coord)
		return
	}
	cell.Unit = load.Unit

	b.addUnitToQueue(load.Unit)
}

func (b *Bot) handlePlaceUnit() {
	unit := b.pickRandomUnitFromHand()
	if unit == nil {
		log.Println("[bot] handlePlaceUnit: no units in hand")
		return
	}

	pos, err := b.randomUnoccupiedSafeZonePos()
	if err != nil {
		log.Printf("[bot] [handlePlaceUnit]: %s", err)
		return
	}

	b.placeUnitAt(unit, pos)
	b.addUnitToQueue(unit)
	b.reply(api.UnitPlacedAction, game.UnitPlacedPayload{
		Coord:      pos,
		TemplateID: unit.TemplateID,
	})
}

func (b *Bot) handlePlayUnit(data json.RawMessage) {
	var load game.PlayUnitPayload
	if err := json.Unmarshal(data, &load); err != nil {
		log.Printf("[bot] [play_unit] unmarshal error: %s", err)
		return
	}

	u := b.unitByID(load.UnitID)
	if u == nil {
		log.Printf("[bot] [play_unit] unit %s not found", load.UnitID)
		b.reply(api.EndTurnAction, nil)
		return
	}

	act := b.simulateUnitTurn(u)
	if act == nil {
		b.reply(api.EndTurnAction, nil)
		return
	}

	if act.moveUnit != nil {
		b.placeUnitAt(u, *act.moveUnit)
		b.reply(api.UnitMovedAction, game.UnitMovedPayload{
			UnitID: u.ID,
			Coord:  *act.moveUnit,
		})
	}

	if act.useAbility != nil {
		b.reply(api.UseAbilityAction, game.UseAbilityPayload{
			UnitID:    u.ID,
			AbilityID: act.useAbility.abilityID,
			Target:    act.useAbility.target,
		})
	}

	b.reply(api.EndTurnAction, nil)
}

func (b *Bot) handleOpponentUnitMoved(data json.RawMessage) {
	var load game.UnitMovedPayload
	if err := json.Unmarshal(data, &load); err != nil {
		log.Printf("[bot] [unit_moved] unmarshal error: %s", err)
		return
	}

	u := b.unitByID(load.UnitID)
	if u == nil {
		log.Printf("[bot] [unit_moved] unit %s not found", load.UnitID)
		return
	}

	b.placeUnitAt(u, load.Coord)
}

func (b *Bot) handleApplyState(data json.RawMessage) {
	var load []game.ApplyState
	if err := json.Unmarshal(data, &load); err != nil {
		log.Printf("[bot] [apply_state] unmarshal error: %s", err)
		return
	}

	for _, st := range load {
		if st.MoveTo != nil {
			b.moveUnit(st.ToUnitID, *st.MoveTo)
		}
		if st.IsDead {
			b.killUnit(st.ToUnitID)
		}
		if st.SetPhantomAP != nil {
			b.state.player.PhantomAP = *st.SetPhantomAP
		}

		if st.ToUnitID.IsZero() {
			continue
		}

		u := b.unitByID(st.ToUnitID)
		if u == nil {
			log.Printf("[bot] [apply_state] unit %s not found", st.ToUnitID)
			continue
		}

		if st.SetHP != nil {
			u.BaseHP = *st.SetHP
		}
		if st.SetBaseHP != nil {
			u.BaseHP = *st.SetBaseHP
		}
		if st.SetAP != nil {
			u.CurrentAP = *st.SetAP
		}
		if st.SetMP != nil {
			u.CurrentMP = *st.SetMP
		}
		if st.SetShield != nil {
			u.CurrentShield = *st.SetShield
		}
		if st.SetAtk != nil {
			u.CurrentAtk = *st.SetAtk
		}
		if st.SetCooldown != nil {
			for abID, cd := range *st.SetCooldown {
				u.SetCooldown(abID, cd)
			}
		}
		if st.AddStatus != nil {
			b.addUnitStatus(u, *st.AddStatus, st.AddStatusMeta)
		}
		if st.RemoveStatus != nil {
			b.removeUnitStatus(u, *st.RemoveStatus)
		}

		if st.SetStatusDuration != nil {
			b.updateUnitStatusDuration(u, st.SetStatusDuration)
		}
	}
}

func (b *Bot) handleNewRound() {
	for _, u := range b.state.queue {
		u.CurrentAP = u.BaseAP
		u.CurrentMP = u.BaseMP
	}
}

func (b *Bot) handleGameOver(action api.Action) {
	switch action {
	case api.YouWinAction:
		log.Println("[bot] won!")
	case api.YouLoseAction:
		log.Println("[bot] lost")
	case api.OppSurrenderedAction:
		log.Println("[bot] opponent surrendered")
	}
}

func (b *Bot) handleServerError(data json.RawMessage) {
	log.Printf("[bot] server error: %s", data)
}
