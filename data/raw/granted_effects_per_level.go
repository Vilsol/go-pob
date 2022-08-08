package raw

type GrantedEffectsPerLevel struct {
	AttackSpeedMultiplier      int   `json:"AttackSpeedMultiplier"`
	AttackTime                 int   `json:"AttackTime"`
	Cooldown                   int   `json:"Cooldown"`
	CooldownBypassType         int   `json:"CooldownBypassType"`
	CooldownGroup              int   `json:"CooldownGroup"`
	CostAmounts                []int `json:"CostAmounts"`
	CostMultiplier             int   `json:"CostMultiplier"`
	CostTypes                  []int `json:"CostTypes"`
	GrantedEffect              int   `json:"GrantedEffect"`
	Level                      int   `json:"Level"`
	LifeReservationFlat        int   `json:"LifeReservationFlat"`
	LifeReservationPercent     int   `json:"LifeReservationPercent"`
	ManaReservationFlat        int   `json:"ManaReservationFlat"`
	ManaReservationPercent     int   `json:"ManaReservationPercent"`
	PlayerLevelReq             int   `json:"PlayerLevelReq"`
	SoulGainPreventionDuration int   `json:"SoulGainPreventionDuration"`
	StoredUses                 int   `json:"StoredUses"`
	VaalSouls                  int   `json:"VaalSouls"`
	VaalStoredUses             int   `json:"VaalStoredUses"`
	Key                        int   `json:"_key"`
}

var GrantedEffectsPerLevels []*GrantedEffectsPerLevel

var grantedEffectsPerLevelsByIDMap map[int]map[int]*GrantedEffectsPerLevel

func InitializeGrantedEffectsPerLevels(version string) error {
	if err := InitHelper(version, "GrantedEffectsPerLevel", &GrantedEffectsPerLevels); err != nil {
		return err
	}

	grantedEffectsPerLevelsByIDMap = make(map[int]map[int]*GrantedEffectsPerLevel)
	for _, i := range GrantedEffectsPerLevels {
		if _, ok := grantedEffectsPerLevelsByIDMap[i.GrantedEffect]; !ok {
			grantedEffectsPerLevelsByIDMap[i.GrantedEffect] = make(map[int]*GrantedEffectsPerLevel)
		}

		grantedEffectsPerLevelsByIDMap[i.GrantedEffect][i.Level] = i
	}

	return nil
}

func (l *GrantedEffectsPerLevel) GetCostTypes() []*CostType {
	if l.CostTypes == nil {
		return nil
	}

	out := make([]*CostType, len(l.CostTypes))
	for i, costType := range l.CostTypes {
		out[i] = CostTypes[costType]
	}
	return out
}
