package ability

type ID string

type Type int

const (
	Skill Type = iota + 1
	Spell
)

const (
	BasicMeleeAttack ID = "basic_melee_attack"
	BasicRangeAttack ID = "basic_range_attack"
	BasicMagicAttack ID = "basic_magic_attack"

	// Tank
	ShieldWall ID = "shield_wall"
	Provoke    ID = "provoke"
	ShieldBash ID = "shield_bash"
	LastStand  ID = "last_stand"

	// warrior
	BattleCry ID = "battle_cry"
	Cleave    ID = "cleave"
	Slam      ID = "slam"
	Frenzy    ID = "frenzy"

	// ranger
	PiercingShot ID = "piercing_shot"
	Prey         ID = "prey"
	Disengage    ID = "disengage"
	CoverFire    ID = "cover_file"

	// rogue
	ShadowStep  ID = "shadow_step"
	Backstab    ID = "backstab"
	Eliminate   ID = "eliminate"
	Opportunity ID = "opportunity"

	// mage
	Translocation ID = "translocation"
	TimeWarp      ID = "time_warp"
	Enfeeble      ID = "enfeeble"
	Meditation    ID = "meditation"

	// support
	Heal        ID = "heal"
	BalanceLife ID = "balance_life"
	Cleanse     ID = "cleanse"
	Renewal     ID = "renewal"
)

// TODO abilities for next
// 1. haste: buff movement
// 2. ressurect
// 3. raize skeleton (counters ressurect)
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

type Ability struct {
	ID          ID     `json:"id"`
	Type        Type   `json:"type"`
	IsPassive   bool   `json:"is_passive"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Range       int    `json:"range"`
	Cooldown    int    `json:"cooldown"`
}

var Abilities = map[ID]Ability{
	BasicMeleeAttack: {
		Name:        "Strike",
		Description: "Delivers a swift, direct blow to a nearby enemy.",
		Range:       1,
		Cooldown:    0,
	},
	BasicRangeAttack: {
		Name:        "Shoot",
		Description: "Fires a deadly projectile at a distant target.",
		Range:       3,
		Cooldown:    0,
	},
	BasicMagicAttack: {
		Name:        "Blast",
		Description: "Hurls a concentrated bolt of arcane energy.",
		Range:       3,
		Cooldown:    0,
	},

	// --- TANK ---
	ShieldWall: {
		Type:        Skill,
		Name:        "Shield Wall",
		Description: "You and adjacent allies gain +3 Shield.",
		Range:       1, Cooldown: 3,
	},
	Provoke: {
		Type:        Skill,
		Name:        "Provoke",
		Description: "Forces target enemies to attack you on their next turn.",
		Range:       3, Cooldown: 3,
	},
	ShieldBash: {
		Type:        Skill,
		Name:        "Shield Bash",
		Description: "Stuns the target for 1 turn.",
		Range:       1, Cooldown: 3,
	},
	LastStand: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Last Stand",
		Description: "If HP falls below 1, gain +3 Shield instead of dying.",
		Range:       0, Cooldown: 5,
	},

	// --- WARRIOR ---
	BattleCry: {
		Type:        Skill,
		Name:        "Battle Cry",
		Description: "Grant +2 Attack to adjacent allies.",
		Range:       1, Cooldown: 3,
	},
	Cleave: {
		Type:        Skill,
		Name:        "Cleave",
		Description: "Attack all enemies in front of you.",
		Range:       1, Cooldown: 3,
	},
	Slam: {
		Type:        Skill,
		Name:        "Slam",
		Description: "Removes all shields from the target.",
		Range:       1, Cooldown: 3,
	},
	Frenzy: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Frenzy",
		Description: "+2 Attack if current HP is below 4.",
		Range:       0, Cooldown: 0,
	},

	// --- RANGER ---
	PiercingShot: {
		Type:        Skill,
		Name:        "Piercing Shot",
		Description: "Fires a shot that passes through all enemies in a line.",
		Range:       3, Cooldown: 3,
	},
	Prey: {
		Type:        Skill,
		Name:        "Prey",
		Description: "Deals 2 damage and marks target for 3 turns. Allies deal +2 damage to marked target.",
		Range:       3, Cooldown: 4,
	},
	Disengage: {
		Type:        Skill,
		Name:        "Disengage",
		Description: "Retreat 2 cells back, breaking engagement.",
		Range:       0, Cooldown: 3,
	},
	CoverFire: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Cover Fire",
		Description: "Counter-attacks enemies that strike allies within your range.",
		Range:       3, Cooldown: 0,
	},

	// --- ROGUE ---
	ShadowStep: {
		Type:        Spell,
		Name:        "Shadow Step",
		Description: "Teleport to target cell and gain +2 Attack for 1 turn.",
		Range:       3, Cooldown: 3,
	},
	Backstab: {
		Type:        Skill,
		Name:        "Backstab",
		Description: "Deals 2x damage if an ally is on the opposite side of the target.",
		Range:       1, Cooldown: 3,
	},
	Eliminate: {
		Type:        Skill,
		Name:        "Eliminate",
		Description: "Deals 3 damage. If target dies, gain 1 AP.",
		Range:       1, Cooldown: 5,
	},
	Opportunity: {
		Type:        Skill,
		IsPassive:   true,
		Name:        "Opportunity",
		Description: "Strikes an enemy if an ally attacks them from melee range.",
		Range:       1, Cooldown: 0,
	},

	// --- MAGE ---
	Translocation: {
		Type:        Spell,
		Name:        "Translocation",
		Description: "Swap places with any unit on the board.",
		Range:       3, Cooldown: 4,
	},
	TimeWarp: {
		Type:        Spell,
		Name:        "Time Warp",
		Description: "Target ally or self gains +1 AP on their next turn.",
		Range:       3, Cooldown: 5,
	},
	Enfeeble: {
		Type:        Spell,
		Name:        "Enfeeble",
		Description: "Reduces target's Attack by 50% for 1 turn.",
		Range:       3, Cooldown: 3,
	},
	Meditation: {
		Type:        Spell,
		IsPassive:   true,
		Name:        "Meditation",
		Description: "If no AP spent, no movement made, and no damage taken this turn: heal 1 HP, gain +1 AP and +1 Movement on next turn.",
		Range:       0, Cooldown: 0,
	},

	// --- SUPPORT ---
	Heal: {
		Type:        Spell,
		Name:        "Heal",
		Description: "Restores 2 HP to target ally.",
		Range:       3, Cooldown: 2,
	},
	BalanceLife: {
		Type:        Spell,
		Name:        "Balance Life",
		Description: "Averages HP of all allies within 2 cells.",
		Range:       2, Cooldown: 5,
	},
	Cleanse: {
		Type:        Spell,
		Name:        "Cleanse",
		Description: "Removes all negative status effects from target ally.",
		Range:       3, Cooldown: 2,
	},
	Renewal: {
		Type:        Spell,
		IsPassive:   true,
		Name:        "Aura of Renewal",
		Description: "If no damage taken this turn, heal self and all adjacent allies for 1 HP at the start of your next turn.",
		Range:       1, Cooldown: 0,
	},
}

func ByID(id ID) Ability {
	s, ok := Abilities[id]
	if ok {
		s.ID = id
	}

	return s
}
