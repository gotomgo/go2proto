package main

import "github.com/iancoleman/strcase"

type protoHelper struct{}

func NewProtoHelper() LanguageHelper {
	return &protoHelper{}
}

func (helper *protoHelper) OnMessage(pi *PackageInfo, m *Message) {}

func (helper *protoHelper) OnField(pi *PackageInfo, f *Field) {
	f.Name = toProtoFieldName(f.Field.Name())
	f.TypeName = toProtoFieldTypeName(f.Field, pi.p)
}

func (helper *protoHelper) OnEnumValue(pi *PackageInfo, e *EnumValue) {
	e.Name = strcase.ToScreamingSnake(e.Const.Name())
}

func (helper *protoHelper) OnMap(pi *PackageInfo, m *Map) {

}
