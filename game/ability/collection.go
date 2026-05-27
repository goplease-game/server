package ability

import "github.com/ognev-dev/goplease/game/ability/effect"

const (
	BasicMeleeAttack ID = "basic_melee_attack"
	BasicRangeAttack ID = "basic_range_attack"
	BasicMagicAttack ID = "basic_magic_attack"

	// Tank
	Fortify     ID = "fortify"
	Provoke     ID = "provoke"
	ShieldBash  ID = "shield_bash"
	UndyingWill ID = "undying_will"

	// warrior
	BattleCry   ID = "battle_cry"
	IdolihuSpin ID = "idolihu_spin"
	PowerPush   ID = "power_push"
	Frenzy      ID = "frenzy"

	// ranger
	PiercingShot  ID = "piercing_shot"
	HuntersMark   ID = "hunters_mark"
	HamstringShot ID = "hamstring_shot"
	CoverFire     ID = "cover_fire"

	// rogue
	ShadowStep  ID = "shadow_step"
	GangUp      ID = "gang_up"
	Eliminate   ID = "eliminate"
	Opportunity ID = "opportunity"

	// mage
	Translocation ID = "translocation"
	TimeWarp      ID = "time_warp"
	Purge         ID = "purge"
	ArcaneChaos   ID = "arcane_chaos"

	// support
	Heal           ID = "heal"
	Equalize       ID = "equalize"
	Purify         ID = "purify"
	BottomlessVial ID = "bottomless_vial"
)

// TODO abilities for next iteration
// 1. haste: buff movement
// 2. resurrect
// 3. raize skeleton (counters resurrect)
// 4. push / pull
// 5. chain lighting / chain heal, etc
// 6. DoT & HoT (poison, bleed, regen, etc)
// 7. AoE (volley of arrows)
// 8. traps
// 9. invisiblity
// 10. clone
// 11. cooldown decrease/increase
// 12. CC (blind, root, etc)
// 13. Silence & disarm
// 14. Thorns
// 15. Life steal
// 16. Alter unit queue

var Abilities = map[ID]Ability{
	BasicMeleeAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Strike",
		Description: "Delivers a direct blow to a nearby enemy.",
		Cooldown:    0,
		Range:       1,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		Effects:     Effects(effect.NewBasicAttack()),
	},
	BasicRangeAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Shoot",
		Description: "Fires a projectile at a distant target.",
		Cooldown:    0,
		Range:       4,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		Effects:     Effects(effect.NewBasicAttack()),
	},
	BasicMagicAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Arcane Bolt",
		Description: "Hurls a bolt of arcane energy.",
		Cooldown:    0,
		Range:       4,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		Effects:     Effects(effect.NewBasicAttack()),
	},

	// --- TANK ---
	Fortify: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Fortify",
		Description: "You and adjacent allies gain +3 Shield. Shield decays by 1 at the start of each turn.",
		Cooldown:    3,
		Range:       0,
		Activation:  Instant,
		TargetMode:  TargetAlliesAndSelf,
		Area:        AreaCircle,
		AreaRadius:  2,
		Effects:     Effects(effect.NewStatusEffect(effect.DecayingShield)),
	},
	Provoke: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Provoke",
		Description: "Forces enemies to attack you on their turn.",
		Cooldown:    3,
		Range:       2,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  2,
		Effects:     Effects(effect.NewStatusEffect(effect.Provoked)),
	},
	ShieldBash: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Shield Bash",
		Description: "Stuns an enemy, preventing their next action.",
		Cooldown:    3,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewStatusEffect(effect.Stun)),
	},
	UndyingWill: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Undying Will",
		Description: "When receiving fatal damage, prevent death: set HP to 1 and gain 3 Shield.",
		Cooldown:    5,
		Range:       0,
		// TODO
		//Effects:     ...
	},

	// --- WARRIOR ---
	BattleCry: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Battle Cry",
		Description: "Nearby allies gain +3 Attack. The bonus decays by 1 at the start of each turn.",
		Cooldown:    3,
		TargetMode:  TargetAllies,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  2,
		Effects:     Effects(effect.NewStatusEffect(effect.DecayingAttack)),
	},
	IdolihuSpin: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "IDOLIHU! Spin",
		Description: "Strikes all adjacent enemies in a single sweeping motion.",
		Cooldown:    3,
		TargetMode:  TargetEnemies,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  1,
		Effects:     Effects(effect.NewBasicAttack()),
	},
	PowerPush: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Power Push",
		Description: "Deals 3 damage and pushes the target back 1 tile. If the target cannot be pushed, deals 5 damage instead.",
		Cooldown:    3,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewAttack(3), effect.NewMove(effect.MovePush, 1)),
		// TODO Custom handler (Attack(5) if cannot move)
	},
	Frenzy: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Frenzy",
		Description: "Gain +2 Attack if there are 2 or more enemies within 2 tiles.",
		// TODO event bus
	},

	// --- RANGER ---
	PiercingShot: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Piercing Shot",
		Description: "Fires a piercing shot that deals attack damage to all enemies in a straight line.",
		Cooldown:    3,
		TargetMode:  TargetEnemies,
		Activation:  SelectAny,
		Area:        AreaLine,
		AreaRadius:  4,
		Effects:     Effects(effect.NewBasicAttack()),
	},
	HuntersMark: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Hunter's Mark",
		Description: "Marks the target for 3 turns. Allies deal +1 damage to the marked target.",
		Cooldown:    4,
		Range:       3,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewStatusEffect(effect.Exposed)),
	},
	HamstringShot: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Hamstring Shot",
		Description: "Deals 2 damage and reduces the target's Move Range to 1 until the end of its next turn.",
		Cooldown:    3,
		Range:       3,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewAttack(2), effect.NewStatusEffect(effect.Hamstrung)),
	},
	CoverFire: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Cover Fire",
		Description: "Once per turn, counter-attacks the first enemy that strikes an ally within your range, dealing 3 flat damage.",
		Range:       3,
		// TODO EventHandler
		//Effects:     nil,
	},

	// --- ROGUE ---
	ShadowStep: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Shadow Step",
		Description: "Teleport to a target cell and gain +2 Attack until the end of your next turn.",
		Cooldown:    3,
		Range:       4,
		Activation:  SelectFreeCell,
		Effects:     Effects(effect.NewMoveTo(), effect.NewStatusEffect(effect.Sharpened)),
	},
	GangUp: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Gang Up",
		Description: "Executes a melee attack. Deals +2 damage if an ally is on the opposite side of the target.",
		Cooldown:    3,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     nil, // TODO AbilityHandler
	},
	Eliminate: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Eliminate",
		Description: "Deals 3 damage. If this attack kills the target, gain 1 AP.",
		Cooldown:    5,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewAttack(3)), // TODO
	},
	Opportunity: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Opportunity",
		Description: "Once per turn, attacks an adjacent enemy when an ally hits them with a melee attack.",
		Cooldown:    0,
		Range:       1,
		Effects:     nil, // TODO
	},

	// --- MAGE ---
	Translocation: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Translocation",
		Description: "Swap places with any ally or enemy within range.",
		Cooldown:    4,
		Range:       3,
		TargetMode:  TargetAny,
		Activation:  SelectAnyUnit,
		Effects:     Effects(effect.NewMoveSwap()),
	},
	TimeWarp: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Time Warp",
		Description: "Target ally or self gains +1 AP. At the end of that unit’s turn, its HP, Shield, and position are restored to their state at the start of the turn.",
		Cooldown:    5,
		Range:       3,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAlly,
		Effects:     nil, // TODO
	},
	Purge: {
		ID:          "",
		Type:        Spell,
		IsPassive:   false,
		Name:        "Purge",
		Description: "Removes all positive effects from target enemy.",
		Cooldown:    3,
		Range:       3,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effects:     Effects(effect.NewDispelPositive()),
	},
	ArcaneChaos: {
		Type:        Spell,
		IsPassive:   true,
		Name:        "Arcane Chaos",
		Description: "At the end of your turn, gain bonuses based on actions taken during the turn:\n- If you did not move: gain +1 Movement Range next turn\n- If no enemies were within 3 tiles: gain +1 Attack Range next turn\n- If you took no damage: restore 1 HP next turn\n- If you took damage: gain 1 Shield\n\nIf 3  conditions are met, also gain +1 Attack next turn.",
		Effects:     nil, // TODO
	},

	// --- SUPPORT ---
	Heal: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Heal",
		Description: "Restores 4 HP to the target ally or self.",
		Cooldown:    1,
		Range:       3,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAllyOrSelf,
		Effects:     Effects(effect.NewHeal(4)),
	},
	Equalize: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Equalize",
		Description: "Equalizes the HP of all allied units within 3 tiles, setting each to the average HP of the affected units.",
		Cooldown:    4,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  3,
		Effects:     nil, // TODO
	},
	Purify: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Purify",
		Description: "Removes all negative status effects from the target ally or self, restores 2 HP, and grants immunity to new debuffs for 1 turn.",
		Cooldown:    3,
		Range:       3,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAllyOrSelf,
		Effects: Effects(
			effect.NewDispelNegative(),
			effect.NewHeal(2),
			effect.NewStatusEffect(effect.DebuffWard),
		),
	},
	BottomlessVial: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Bottomless Vial",
		Description: "The first time each turn July loses HP, her maximum HP permanently increases by 1.",
		Effects:     nil, // TODO
	},
}
