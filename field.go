package main

import "go/types"

type Field struct {
	Name       string
	TypeName   string
	Order      int
	IsRepeated bool
	JSONName   string
	Field      *types.Var
}
