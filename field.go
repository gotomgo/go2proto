package main

import "go/types"

type Field struct {
	Field      *types.Var
	Name       string
	TypeName   string
	Order      int
	IsRepeated bool
	JSONName   string
}
