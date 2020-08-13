package main

import (
	"go/types"
	"sort"
	"strconv"
)

type Enum struct {
	Name           string
	Values         []*EnumValue
	AllowAlias     bool
	MissingDefault bool
	HasGaps        bool
	Enum           types.Object
}

func NewEnum(t types.Object) *Enum {
	return &Enum{Name: getEnumTypeName(t), Enum: t}
}

func getEnumTypeName(t types.Object) string {
	return splitTypeNameHelper(t.Type())
}

// isEnumType determines if the Underlying type is a supported enum type
func isEnumType(t types.Object) (result bool) {
	// should be of type int (or int64?)
	baseType := t.Type().Underlying().String()

	switch baseType {
	case GoTypeInt, GoTypeInt32, GoTypeInt64:
		result = true
	}

	return
}

func (e *Enum) AddValue(c *types.Const) (result *EnumValue, err error) {
	var val int64

	// convert the enum value to int64
	val, err = strconv.ParseInt(c.Val().String(), 0, 64)
	if err == nil {
		result = NewEnumValue(c, val)
		// add the value to the enum type
		e.Values = append(e.Values, result)
	}

	return
}

func (e *Enum) Canonicalize() {
	// sort enum values by value
	sort.Slice(e.Values, func(i, j int) bool {
		if e.Values[i].Value == e.Values[j].Value {
			return e.Values[i].Name < e.Values[j].Name
		}
		return e.Values[i].Value < e.Values[j].Value
	})

	// assume zero is not defined
	e.MissingDefault = true

	var lastValue int64
	var pLastValue *int64

	for _, val := range e.Values {
		if val.Value == 0 {
			e.MissingDefault = false
		}

		if (pLastValue != nil) && (val.Value == *pLastValue) {
			e.AllowAlias = true

			if val.Value > (*pLastValue + 1) {
				e.HasGaps = true
			}
		}

		lastValue = val.Value
		pLastValue = &lastValue
	}
}

type EnumValue struct {
	Const *types.Const
	Name  string
	Value int64
}

// NewEnumValue creates an instance of EnumValue based on a const
func NewEnumValue(c *types.Const, value int64) *EnumValue {
	return &EnumValue{Value: value, Const: c}
}
