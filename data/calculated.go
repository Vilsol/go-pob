package data

func DamageStatsForType(t int) []string {
	out := make([]string, 1)
	out[0] = "Damage"
	for damageType, flag := range DamageTypeFlags {
		if t&flag != 0 {
			out = append(out, string(damageType)+"Damage")
		}
	}
	return out
}
