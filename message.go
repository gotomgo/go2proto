package main

import (
	"go/types"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/packages"
)

type Message struct {
	Struct *types.Struct
	Name   string
	Fields []*Field
}

func newMessage(name string, s *types.Struct) *Message {
	return &Message{
		Name:   name,
		Struct: s,
		Fields: []*Field{},
	}
}

func CreateMessage(t types.Object, s *types.Struct, p *packages.Package) *Message {
	msg := newMessage(t.Name(), s)

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
			Field:      f,
			IsRepeated: isRepeated(f),
			Order:      i + 1,
			JSONName:   jsonName,
		}
		msg.Fields = append(msg.Fields, newField)
	}

	return msg
}
