package calculator

import "go-pob/utils"

type DamageType string

const (
	DamageTypeCold      = DamageType("Cold")
	DamageTypeLightning = DamageType("Lightning")
	DamageTypeFire      = DamageType("Fire")
)

type NonDamagingAilment string

const (
	NDAChill   = NonDamagingAilment("Chill")
	NDAFreeze  = NonDamagingAilment("Freeze")
	NDAShock   = NonDamagingAilment("Shock")
	NDAScorch  = NonDamagingAilment("Scorch")
	NDABrittle = NonDamagingAilment("Brittle")
	NDASap     = NonDamagingAilment("Sap")
)

type NonDamagingAilmentData struct {
	AssociatedType DamageType
	Alt            bool
	Default        *float64
	Min            float64
	Max            float64
	Precision      float64
	Duration       *float64
}

var nonDamagingAilments = map[NonDamagingAilment]NonDamagingAilmentData{
	NDAChill: {
		AssociatedType: DamageTypeCold,
		Alt:            false,
		Default:        utils.Ptr[float64](10),
		Min:            5,
		Max:            30,
		Precision:      0,
		Duration:       utils.Ptr[float64](2),
	},
	NDAFreeze: {
		AssociatedType: DamageTypeCold,
		Alt:            false,
		Default:        nil,
		Min:            0.3,
		Max:            3,
		Precision:      2,
		Duration:       nil,
	},
	NDAShock: {
		AssociatedType: DamageTypeLightning,
		Alt:            false,
		Default:        utils.Ptr[float64](15),
		Min:            5,
		Max:            50,
		Precision:      0,
		Duration:       utils.Ptr[float64](2),
	},
	NDAScorch: {
		AssociatedType: DamageTypeFire,
		Alt:            true,
		Default:        utils.Ptr[float64](10),
		Min:            0,
		Max:            30,
		Precision:      0,
		Duration:       utils.Ptr[float64](4),
	},
	NDABrittle: {
		AssociatedType: DamageTypeCold,
		Alt:            true,
		Default:        utils.Ptr[float64](5),
		Min:            0,
		Max:            15,
		Precision:      2,
		Duration:       utils.Ptr[float64](4),
	},
	NDASap: {
		AssociatedType: DamageTypeLightning,
		Alt:            true,
		Default:        utils.Ptr[float64](6),
		Min:            0,
		Max:            20,
		Precision:      0,
		Duration:       utils.Ptr[float64](4),
	},
}
