package calculator

// crystalline:promise
func (c *Calculator) BuildOutput(mode OutputMode) *Environment {
	env, _, _, _ := InitEnv(c.PoB, mode)
	PerformCalc(env)
	return env
}
