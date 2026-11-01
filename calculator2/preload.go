package calculator2

// Preload runs the lazy game-data loads every calculation reads; its error is the one a calculation returns. Minion
// data is left to the calculations that read it: most builds never do, and its load blocks the worker for seconds.
func Preload() error {
	initializeParser()
	gemLookup()
	gemByBaseName("")
	if _, err := loadSkillData(); err != nil {
		return err
	}
	_, err := costDivisors()
	return err
}
