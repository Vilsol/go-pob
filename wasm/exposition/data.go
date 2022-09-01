package exposition

import "github.com/Vilsol/go-pob/data/raw"

func GetStatByIndex(id int) *raw.Stat {
	return raw.Stats[id]
}
