package calculator

import "testing"

// TestAilmentStackConfigRowsDispatch pins the four enemy-ailment config rows that had no handler.
func TestAilmentStackConfigRowsDispatch(t *testing.T) {
	for _, name := range []string{
		"ShockStacks",
		"ScorchStacks",
		"HoarfrostStacks",
		"conditionEnemyOnBrineGround",
	} {
		assertConfigDispatchable(t, name)
	}
	for _, name := range []string{"ShockStacks", "ScorchStacks"} {
		if !isConfigPlaceholderVar(name) {
			t.Errorf("%s declares defaultPlaceholderState upstream but is absent from "+
				"configPlaceholderVars — its handler can never fire off the placeholder", name)
		}
	}
}
