package exposition

import (
	"github.com/Vilsol/go-pob-data/poe"
)

func GetStatByIndex(id int) *poe.Stat {
	return poe.Stats[id]
}
