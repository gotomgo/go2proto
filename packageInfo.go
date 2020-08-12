package main

import (
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"
)

// PackageInfo stores high-level information about the types in a package
type PackageInfo struct {
	p    *packages.Package
	seen map[string]bool

	Name     string
	Path     string
	Messages []*Message
	Imports  []string
	Enums    map[string]*Enum
	Maps     map[string]*Map
}

// NewPackageInfo creates an instance of PackageInfo which stores information
// about the types in the package
func NewPackageInfo(p *packages.Package) PackageInfo {
	return PackageInfo{
		p:     p,
		Name:  p.Name,
		Path:  p.PkgPath,
		seen:  map[string]bool{},
		Enums: map[string]*Enum{},
		Maps:  map[string]*Map{},
	}
}

// Canonicalize sorts and finalizes the types from this package
func (pi *PackageInfo) Canonicalize() {
	pi.canonicalizeEnums()
	pi.canonicalizeMessages()
}

// IsPackageType determines if a type is defined in this package
func (pi *PackageInfo) IsPackageType(t types.Object) bool {
	if t == nil {
		return false
	}

	return getPackageFromType(t.Type().String()) == pi.p.PkgPath
}

// GetEnum returns an *enum for the specified enum type name
//
//  Notes
//    If an enum with enumTypeName does not exist, it is created
//    and added to the known enums for this package
//
//    enumTypeName is the simple type name. Caller should ensure that
//    it represents a type declared in this package
//
func (pi *PackageInfo) GetEnum(enumTypeName string, t types.Object) (result *Enum) {
	// have we already seen it?
	result, ok := pi.Enums[enumTypeName]

	// If not, create the enum type and remember it
	if !ok {
		result = &Enum{Name: enumTypeName, Enum: t}
		pi.Enums[enumTypeName] = result
	}

	return
}

// isEnumType determines if the Underlying type is a supported enum type
func isEnumType(t types.Object) (result bool) {
	// should be of type int (or int64?)
	baseType := t.Type().Underlying().String()

	switch baseType {
	case "int", "int32", "int64":
		result = true
	}

	return
}

// GetEnumForType will get/create a *Enum if t is const, a supported underlying
// type, and declared in this package
func (pi *PackageInfo) GetEnumForType(t types.Object) (result *Enum) {
	// enums are always derived from constants
	if _, ok := t.(*types.Const); ok {
		// should be of type int (or int64?)
		// the enum type for the const must be defined in the same package
		if pi.IsPackageType(t) && isEnumType(t) {
			// enum Type Name
			enumTypeName := splitTypeNameHelper(t.Type())

			result = pi.GetEnum(enumTypeName, t)
		}
	}

	return
}

func (pi *PackageInfo) canonicalizeEnums() {
	for _, enum := range pi.Enums {
		enum.Canonicalize()
	}
}

func (pi *PackageInfo) canonicalizeMessages() {
	// sort messages (structs) by name
	sort.Slice(pi.Messages, func(i, j int) bool { return pi.Messages[i].Name < pi.Messages[j].Name })
}

// IsMessage determines if the type is a struct that is defined in this package
// and has not been previously processed
func (pi *PackageInfo) IsMessage(t types.Object) bool {
	if _, ok := t.Type().Underlying().(*types.Struct); !ok {
		return false
	}

	if _, ok := pi.seen[t.Type().String()]; ok {
		return false
	}

	pi.seen[t.Type().String()] = true

	typePkgName := getPackageFromType(t.Type().String())
	if typePkgName != t.Pkg().Path() {
		pi.Imports = append(pi.Imports, typePkgName)
		return false
	}

	return true
}

func (pi *PackageInfo) AddMap(t types.Object) (result *Map) {
	m, ok := t.Type().Underlying().(*types.Map)
	if !ok {
		return nil
	}

	if _, ok = pi.Maps[t.Name()]; ok {
		return nil
	}

	result = &Map{
		Name:               t.Name(),
		KeyType:            m.Key().String(),
		ElemType:           m.Elem().String(),
		UnderlyingKeyType:  m.Key().Underlying().String(),
		UnderlyingElemType: m.Elem().Underlying().String(),
		Map:                m,
	}

	pi.Maps[t.Name()] = result

	return result
}
