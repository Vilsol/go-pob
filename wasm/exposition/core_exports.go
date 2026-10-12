package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func Core30Exports(r bind.Registry) {
	r.Func(ItemPrimarySlot, bind.AsPromise())
	r.Type(calculator.SecondaryAscendOption{}, bind.Plain())
	r.Type(calculator.SecondaryAscendancyDropdown{}, bind.Plain())
	r.Func(SecondaryAscendancies, bind.AsPromise())
	r.Func(ChangeSecondaryAscendancy, bind.AsPromise())
	r.Type(calculator.GroupSourceName{}, bind.Plain())
	r.Func(SocketGroupSourceNames, bind.AsPromise())
	r.Func(OptimiseSockets, bind.AsPromise())
}

func ItemPrimarySlot(build *pob.PathOfBuilding, id int, kind string, name string) string {
	return calculator.DropItemPrimarySlot(build, id, calculator.ItemDBKind(kind), name)
}

func SecondaryAscendancies(build *pob.PathOfBuilding) calculator.SecondaryAscendancyDropdown {
	return calculator.SecondaryAscendancyList(build)
}

func ChangeSecondaryAscendancy(build *pob.PathOfBuilding, ascendClassID int) {
	calculator.SelectSecondaryAscendClass(build, ascendClassID)
}

func SocketGroupSourceNames(build *pob.PathOfBuilding, setIndex int, groupIndex int) []calculator.GroupSourceName {
	return calculator.SocketGroupSourceNames(build, setIndex, groupIndex)
}

func OptimiseSockets(build *pob.PathOfBuilding, slotName string) bool {
	return calculator.OptimiseSockets(build, slotName)
}
