package mod

import (
	"log/slog"
	"reflect"
)

type ModValueMultiType string

const (
	ModValueMultiTypeFloat = ModValueMultiType("float")
	ModValueMultiTypeFlag  = ModValueMultiType("flag")
	ModValueMultiTypeList  = ModValueMultiType("list")
)

type ModValueMulti struct {
	valueType ModValueMultiType

	ValueFloat float64
	ValueFlag  bool
	ValueList  any // TODO: Find a better data type for these mods
}

func (m *ModValueMulti) Type() ModValueMultiType {
	return m.valueType
}

func (m *ModValueMulti) Float() float64 {
	return m.ValueFloat
}

func (m *ModValueMulti) Flag() bool {
	return m.ValueFlag
}

func (m *ModValueMulti) List() any {
	return m.ValueList
}

func (m *ModValueMulti) SetFloat(v float64) {
	m.ValueFloat = v
}

func (m *ModValueMulti) SetFlag(v bool) {
	m.ValueFlag = v
}

func (m *ModValueMulti) SetList(v any) {
	m.ValueList = v
}

func (m *ModValueMulti) Clone() *ModValueMulti {
	listValue := m.ValueList
	if listValue != nil {
		cloneable, ok := listValue.(Cloneable)
		if ok {
			listValue = cloneable.Clone(reflect.TypeOf(m.ValueList).Kind() == reflect.Ptr)
		} else {
			slog.Warn("list type not cloneable", slog.Any("type", reflect.TypeOf(listValue)))
		}
	}

	return &ModValueMulti{
		valueType:  m.valueType,
		ValueFloat: m.ValueFloat,
		ValueFlag:  m.ValueFlag,
		ValueList:  listValue,
	}
}

func NewModValueFloat(v float64) *ModValueMulti {
	return &ModValueMulti{
		valueType:  ModValueMultiTypeFloat,
		ValueFloat: v,
	}
}

func NewModValueFlag(v bool) *ModValueMulti {
	return &ModValueMulti{
		valueType: ModValueMultiTypeFlag,
		ValueFlag: v,
	}
}

func NewModValueList(v any) *ModValueMulti {
	return &ModValueMulti{
		valueType: ModValueMultiTypeList,
		ValueList: v,
	}
}
