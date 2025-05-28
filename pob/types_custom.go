package pob

type GameVersion string

type TreeVersion string

const (
	GameVersion3_0 = GameVersion("3_0")
)

const LiveTargetVersion = GameVersion3_0

type AscendancyID int

const (
	Ascendant AscendancyID = iota
	Assassin
	Berserker
	Champion
	Chieftain
	Deadeye
	Elementalist
	Gladiator
	Guardian
	Hierophant
	Inquisitor
	Juggernaut
	Necromancer
	Occultist
	Pathfinder
	Raider
	Saboteur
	Slayer
	Trickster
)

type AscendancyName string

var AscendancyNameByID = map[AscendancyID]AscendancyName{
	Ascendant:    "Ascendant",
	Assassin:     "Assassin",
	Berserker:    "Berserker",
	Champion:     "Champion",
	Chieftain:    "Chieftain",
	Deadeye:      "Deadeye",
	Elementalist: "Elementalist",
	Gladiator:    "Gladiator",
	Guardian:     "Guardian",
	Hierophant:   "Hierophant",
	Inquisitor:   "Inquisitor",
	Juggernaut:   "Juggernaut",
	Necromancer:  "Necromancer",
	Occultist:    "Occultist",
	Pathfinder:   "Pathfinder",
	Raider:       "Raider",
	Saboteur:     "Saboteur",
	Slayer:       "Slayer",
	Trickster:    "Trickster",
}

type ClassID int

const (
	Scion ClassID = iota
	Marauder
	Ranger
	Witch
	Duelist
	Templar
	Shadow
)

var ClassAscendancies = map[ClassID][]AscendancyID{
	Scion:    {Ascendant},
	Marauder: {Juggernaut, Berserker, Chieftain},
	Ranger:   {Deadeye, Pathfinder, Raider},
	Witch:    {Elementalist, Necromancer, Occultist},
	Duelist:  {Slayer, Gladiator, Champion},
	Templar:  {Hierophant, Inquisitor, Guardian},
	Shadow:   {Assassin, Trickster, Saboteur},
}

type ClassName string

var ClassNameByID = map[ClassID]ClassName{
	Scion:    "Scion",
	Marauder: "Marauder",
	Ranger:   "Ranger",
	Witch:    "Witch",
	Duelist:  "Duelist",
	Templar:  "Templar",
	Shadow:   "Shadow",
}

type OilType string
