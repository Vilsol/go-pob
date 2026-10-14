package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func ItemRebaseExports(r bind.Registry) {
	r.Type(calculator.RebaseResult{}, bind.Plain())

	r.Func(RebaseChoices, bind.AsPromise())
	r.Func(RebaseItemText, bind.AsPromise())
}

func RebaseChoices(text string) []string {
	return calculator.RebaseChoices(text)
}

func RebaseItemText(text string, baseName string) (calculator.RebaseResult, bool) {
	return calculator.RebaseItemText(text, baseName)
}
