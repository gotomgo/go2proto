package main

type LanguageHelper interface {
	OnMessage(pi *PackageInfo, m *Message)
	OnField(pi *PackageInfo, f *Field)
	OnEnumValue(pi *PackageInfo, e *EnumValue)
	OnMap(pi *PackageInfo, m *Map)
}
