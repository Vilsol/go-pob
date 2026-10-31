package poe2

// MinionTemplate is the state one #emit of src/Export/Scripts/minions.lua reads that does not come from the
// game: the entry key, the MonsterVarieties id, #limit ("" is none), #flags, the #mod lines and the
// skills the #monster line adds after the variety's own.
type MinionTemplate struct {
	File    string
	Key     string
	Variety string
	Limit   string
	Flags   []string
	Mods    []string
	Skills  []string
}
