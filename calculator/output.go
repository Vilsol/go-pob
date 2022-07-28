package calculator

func (c *Calculator) BuildOutput(mode OutputMode) {
	env, _, _, _ := InitEnv(c.PoB, mode)
	PerformCalc(env)
}
