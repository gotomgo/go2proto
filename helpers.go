package main

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/packages"
)

func isStruct(t types.Object) bool {
	_, ok := t.Type().Underlying().(*types.Struct)
	return ok
}

func isConst(t types.Object) bool {
	_, ok := t.(*types.Const)
	return ok
}

func isMap(t types.Object) bool {
	_, ok := t.Type().Underlying().(*types.Map)
	return ok
}

func toProtoFieldTypeName(f *types.Var, p *packages.Package) string {
	switch f.Type().Underlying().(type) {
	case *types.Basic:
		name := f.Type().String()
		return normalizeType(name, p)
	case *types.Slice:
		name := splitNameHelper(f)
		return normalizeType(strings.TrimLeft(name, "[]"), p)

	case *types.Pointer, *types.Struct:
		name := splitNameHelper(f)
		return normalizeType(name, p)
	case *types.Map:
		if m, ok := f.Type().(*types.Map); ok {
			return fmt.Sprintf("map<%s,%s>", normalizeType(m.Key().String(), p), normalizeType(m.Elem().String(), p))
		}
	}
	return f.Type().String()
}

func splitTypeNameHelperStr(typeName string) string {
	// TODO: this is ugly. Find another way of getting field type name.
	parts := strings.Split(typeName, ".")

	name := parts[len(parts)-1]

	if name[0] == '*' {
		name = name[1:]
	}
	return name
}

func splitTypeNameHelper(t types.Type) string {
	// TODO: this is ugly. Find another way of getting field type name.
	parts := strings.Split(t.String(), ".")

	name := parts[len(parts)-1]

	if name[0] == '*' {
		name = name[1:]
	}
	return name
}

func splitNameHelper(f types.Object) string {
	return splitTypeNameHelper(f.Type())
}

func normalizeType(name string, p *packages.Package) (result string) {
	switch name {
	case GoTypeInt, GoTypeInt64, GoTypeInt32:
		result = ProtoTypeInt64
	case GoTypeFloat32:
		result = ProtoTypeFloat
	case GoTypeFloat64:
		result = ProtoTypeDouble
	default:
		pkgName := getPackageFromType(name)
		if pkgName == p.PkgPath {
			result = splitTypeNameHelperStr(name)
		} else {
			result = name
		}
	}

	return
}

func isRepeated(f types.Object) bool {
	_, ok := f.Type().Underlying().(*types.Slice)
	return ok
}

func toProtoFieldName(name string) string {
	if len(name) == 2 {
		return strings.ToLower(name)
	}

	return strcase.ToSnake(name)
}

func getPackageFromType(typeName string) (result string) {
	lastDot := strings.LastIndex(typeName, ".")
	if lastDot >= 0 {
		result = typeName[:lastDot]
	} else {
		result = typeName
	}

	return result
}
