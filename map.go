package main

import (
	"fmt"
	"go/types"
)

type Map struct {
	Name               string
	KeyType            string
	UnderlyingKeyType  string
	ElemType           string
	UnderlyingElemType string
	Map                *types.Map
}

// NewMap creates an instance of *Map from a types.Object whose underlying
// type is *types.Map
func NewMap(t types.Object) *Map {
	m, ok := t.Type().Underlying().(*types.Map)
	if !ok {
		panic(fmt.Errorf("expecting type '%s' to represent a map", t.Name()))
	}

	return &Map{
		Name:               t.Name(),
		Map:                m,
		KeyType:            m.Key().String(),
		ElemType:           m.Elem().String(),
		UnderlyingKeyType:  m.Key().Underlying().String(),
		UnderlyingElemType: m.Elem().Underlying().String(),
	}
}
