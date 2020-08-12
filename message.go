package main

import (
	"go/types"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/packages"
)

type Message struct {
	Name   string
	Fields []*Field
	Struct *types.Struct
}

func NewMessage(name string, s *types.Struct) *Message {
	return &Message{
		Name:   name,
		Fields: []*Field{},
		Struct: s,
	}
}

func createMessage(t types.Object, s *types.Struct, p *packages.Package) *Message {
	msg := NewMessage(t.Name(), s)

	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if !f.Exported() {
			continue
		}

		var jsonName string

		if tags, err := structtag.Parse(s.Tag(i)); err == nil {
			if jsonTag, err := tags.Get("json"); err == nil {
				jsonName = jsonTag.Name
			}
		}

		newField := &Field{
			Name:       toProtoFieldName(f.Name()),
			TypeName:   toProtoFieldTypeName(f, p),
			IsRepeated: isRepeated(f),
			Order:      i + 1,
			JSONName:   jsonName,
			Field:      f,
		}
		msg.Fields = append(msg.Fields, newField)
	}

	return msg
}
