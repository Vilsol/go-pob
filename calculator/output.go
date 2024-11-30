package calculator

var envCache = &EnvironmentCache{}

// crystalline:promise
func (c *Calculator) BuildOutput(mode OutputMode) *Environment {
	env, _, _, _ := InitEnv(c.PoB, envCache, mode)
	PerformCalc(env)
	return env
}
