package main

import "go/types"

type Map struct {
	Name               string
	KeyType            string
	UnderlyingKeyType  string
	ElemType           string
	UnderlyingElemType string
	Map                *types.Map
}
