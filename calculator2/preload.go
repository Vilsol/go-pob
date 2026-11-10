package calculator2

import "github.com/Vilsol/go-pob/data/poe2"

// Preload runs the lazy game-data loads every calculation reads; its error is the one a calculation returns. Minion
// data is left to the calculations that read it: most builds never do, and its load blocks the worker for seconds.
// step, when set, hears each load as it starts, with the count of loads done before it.
func Preload(step func(stage string, done, total int)) error {
	loads := []struct {
		stage string
		load  func() error
	}{
		{"Item tables", poe2.LoadItemTables},
		{"Mod parser", func() error { initializeParser(); return nil }},
		{"Gems", func() error { gemLookup(); gemByBaseName(""); return nil }},
		{"Skill data", func() error { _, err := loadSkillData(); return err }},
		{"Cost divisors", func() error { _, err := costDivisors(); return err }},
	}
	for i, l := range loads {
		if step != nil {
			step(l.stage, i, len(loads))
		}
		if err := l.load(); err != nil {
			return err
		}
	}
	return nil
}
