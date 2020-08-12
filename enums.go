package main

import (
	"go/types"
	"sort"
	"strconv"

	"github.com/iancoleman/strcase"
)

type Enum struct {
	Name           string
	Values         []*EnumValue
	AllowAlias     bool
	MissingDefault bool
	HasGaps        bool
	Enum           types.Object
}

func (e *Enum) AddValue(c *types.Const) (err error) {
	var val int64

	// convert the enum value to int64
	val, err = strconv.ParseInt(c.Val().String(), 0, 64)
	if err == nil {
		// add the value to the enum type
		e.Values = append(e.Values, &EnumValue{
			Name:  strcase.ToScreamingSnake(c.Name()),
			Value: val,
			Const: c,
		})
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
	Name  string
	Value int64
	Const *types.Const
}
