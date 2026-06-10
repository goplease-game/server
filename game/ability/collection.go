package ability

import "github.com/ognev-dev/goplease/game/ability/status"

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
	FocusField    ID = "focus_field"

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
// 7. AoE (volley of arrows, fireball - etc)
// 8. traps
// 9. invisiblity
// 10. clone
// 11. cooldown decrease/increase
// 12. CC (blind, root, etc)
// 13. Silence & disarm
// 14. Thorns
// 15. Life steal
// 16. Alter unit queue
// 17. buff steal, debuff transfer|reflect

const HintCurrentATK = "ATK"

var Abilities = map[ID]Ability{
	BasicMeleeAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Strike",
		Description: "Delivers direct blow to a nearby enemy.",
		Cooldown:    0,
		Range:       1,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		DamageHint:  HintCurrentATK,
	},
	BasicRangeAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Shoot",
		Description: "Fires projectile at a distant target.",
		Cooldown:    0,
		Range:       3,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		DamageHint:  HintCurrentATK,
	},
	BasicMagicAttack: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Arcane Bolt",
		Description: "Hurls bolt of arcane energy.",
		Cooldown:    0,
		Range:       3,
		Activation:  SelectEnemy,
		TargetMode:  TargetEnemies,
		DamageHint:  HintCurrentATK,
	},

	// --- TANK ---
	Fortify: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Fortify",
		Description: "You and adjacent allies gain +5 Shield. Shield decays by 1 at the start of each turn.",
		Cooldown:    2,
		Activation:  Instant,
		TargetMode:  TargetAlliesAndSelf,
		Area:        AreaCircle,
		AreaRadius:  2,
		Effect:      Effect{AddShield: 5},
	},
	Provoke: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Provoke",
		Description: "Forces enemies to attack you on their turn.",
		Cooldown:    2,
		Range:       2,
		TargetMode:  TargetEnemies,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  2,
	},
	ShieldBash: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Shield Bash",
		Description: "Stuns an enemy, preventing their next action.",
		Cooldown:    2,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effect:      Effect{ApplyStatus: status.Stunned},
	},
	UndyingWill: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Undying Will",
		Description: "When receiving fatal damage, prevent death: set HP to 1 and gain 5 Shield.",
		Cooldown:    5,
		Effect:      Effect{HealHP: 1, AddShield: 10},
	},

	// --- WARRIOR ---
	BattleCry: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Battle Cry",
		Description: "Grants +1 Attack to nearby allies for 3 turns",
		Cooldown:    2,
		TargetMode:  TargetAllies,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  2,
		Effect:      Effect{ApplyStatus: status.Rallied},
	},
	IdolihuSpin: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "IDOLIHU! Spin",
		Description: "Strikes all adjacent enemies in a single sweeping motion.",
		Cooldown:    2,
		TargetMode:  TargetEnemies,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  1,
		DamageHint:  HintCurrentATK,
	},
	PowerPush: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Power Push",
		Description: "Deals 2 damage and pushes the target back 1 tile. If the target cannot be pushed, deals 4 damage instead.",
		Cooldown:    2,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		DamageHint:  "2/4",
		Effect:      Effect{DealDamage: 2, DealAltDamage: 4},
	},
	Frenzy: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Frenzy",
		Description: "Gains +1 Attack if there are 2 or more enemies within 2 cells.",
		AreaRadius:  2,
		Effect:      Effect{ApplyStatus: status.Frenzied},
	},

	// --- RANGER ---
	PiercingShot: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Piercing Shot",
		Description: "Fires a piercing shot that deals 3 damage to each enemy in a straight line.",
		Cooldown:    2,
		TargetMode:  TargetEnemies,
		Activation:  SelectAny,
		Area:        AreaLine,
		AreaRadius:  3,
		DamageHint:  "3",
		Effect:      Effect{DealDamage: 3},
	},
	HuntersMark: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Hunter's Mark",
		Description: "Marks target for 3 turns. Allies deal +1 damage to marked target.",
		Cooldown:    2,
		Range:       3,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		Effect:      Effect{ApplyStatus: status.Exposed},
	},
	HamstringShot: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Hamstring Shot",
		Description: "Deals 2 damage and reduces target's Move Range to 1 for next 2 turns",
		Cooldown:    2,
		Range:       3,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		DamageHint:  "2",
		Effect:      Effect{DealDamage: 2, ApplyStatus: status.Hamstrung},
	},
	CoverFire: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Cover Fire",
		Description: "Once per round, counter-attacks the first enemy that strikes an ally within your range, dealing 3 flat damage.",
		Cooldown:    1,
		Range:       3,
		DamageHint:  "3",
		Effect:      Effect{DealDamage: 3},
	},

	// --- ROGUE ---
	ShadowStep: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Shadow Step",
		Description: "Teleport to target cell and gain +1 Attack until the end of your next turn.",
		Cooldown:    2,
		Range:       4,
		Activation:  SelectFreeCell,
		Effect:      Effect{ApplyStatus: status.Sharpened},
	},
	GangUp: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Gang Up",
		Description: "Executes a melee attack. Deals +2 bonus damage if an ally is on the opposite side of the target",
		Cooldown:    2,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		DamageHint:  "ATK/ATK+2",
		Effect:      Effect{BonusDamage: 2},
	},
	Eliminate: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Eliminate",
		Description: "Deals 3 damage. If this attack kills the target, gain 1 AP.",
		Cooldown:    2,
		Range:       1,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
		DamageHint:  "3",
		Effect:      Effect{DealDamage: 3, AddAP: 1},
	},
	Opportunity: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Opportunity",
		Description: "Once per turn, attacks an adjacent enemy when an ally hits them within melee range.",
		Cooldown:    1,
		Range:       1,
		DamageHint:  HintCurrentATK,
	},

	// --- MAGE ---
	Translocation: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Translocation",
		Description: "Swap places with any ally or enemy within range.",
		Cooldown:    2,
		Range:       4,
		TargetMode:  TargetAny,
		Activation:  SelectAnyUnit,
	},
	TimeWarp: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Time Warp",
		Description: "Target ally or self gains +1 AP at the start of their next turn. At the end of that turn, their HP, Shield, and position are restored to their state at the start of the turn.",
		Cooldown:    2,
		Range:       2,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAllyOrSelf,
		Effect:      Effect{ApplyStatus: status.TemporalAnchor},
	},
	Purge: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Purge",
		Description: "Removes all positive effects from target enemy.",
		Cooldown:    2,
		Range:       2,
		TargetMode:  TargetEnemies,
		Activation:  SelectEnemy,
	},
	FocusField: {
		Type:        Spell,
		IsPassive:   true,
		Name:        "Focus Field",
		Description: "All friendly units starting their turn next to Mist have their cooldowns reduced by 1  (excluding passive abilities).",
		Range:       1,
	},

	// --- SUPPORT ---
	Heal: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Heal",
		Description: "Restores 5 HP to the target ally or self.",
		Cooldown:    2,
		Range:       3,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAllyOrSelf,
		Effect:      Effect{HealHP: 5},
	},
	Equalize: {
		Type:        Spell,
		IsPassive:   false,
		Name:        "Equalize",
		Description: "Equalizes the HP of all allied units within 3 tiles, setting each to the average HP of the affected units.",
		Cooldown:    2,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  Instant,
		Area:        AreaCircle,
		AreaRadius:  2,
	},
	Purify: {
		Type:        Skill,
		IsPassive:   false,
		Name:        "Purify",
		Description: "Removes all negative status effects from the target ally or self, restores 2 HP, and grants immunity to new debuffs for 1 turn.",
		Cooldown:    2,
		Range:       2,
		TargetMode:  TargetAlliesAndSelf,
		Activation:  SelectAllyOrSelf,
		Effect:      Effect{HealHP: 2, ApplyStatus: status.DebuffWard},
	},
	BottomlessVial: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Bottomless Vial",
		Description: "The first time each round a friendly unit within 3 cells takes damage, their maximum HP increases by 2 and they restore 1 HP.",
		Cooldown:    1,
		AreaRadius:  3,
		Effect:      Effect{AddHP: 2, HealHP: 1},
	},
}
