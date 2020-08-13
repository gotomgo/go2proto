package main

import (
	"fmt"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"
)

// PackageInfo stores high-level information about the types in a package
type PackageInfo struct {
	p      *packages.Package
	helper LanguageHelper
	seen   map[string]bool

	Name     string
	Path     string
	Messages []*Message
	Imports  []string
	Enums    map[string]*Enum
	Maps     map[string]*Map
}

// NewPackageInfo creates an instance of PackageInfo which stores information
// about the types in the package
func NewPackageInfo(p *packages.Package, helper LanguageHelper) PackageInfo {
	return PackageInfo{
		p:      p,
		helper: helper,
		Name:   p.Name,
		Path:   p.PkgPath,
		seen:   map[string]bool{},
		Enums:  map[string]*Enum{},
		Maps:   map[string]*Map{},
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
		result = NewEnum(t)
		pi.Enums[enumTypeName] = result
	}

	return
}

// GetEnumForType will get/create a *Enum if t is const, a supported underlying
// type, and declared in this package
func (pi *PackageInfo) GetEnumForType(t types.Object) (result *Enum) {
	// enums are always derived from constants
	if _, ok := t.(*types.Const); ok {
		// should be a type used for enums and the enum type for the const must
		// be defined in the same package
		if pi.IsPackageType(t) && isEnumType(t) {
			result = pi.GetEnum(getEnumTypeName(t), t)
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
func (pi *PackageInfo) shouldAddMessage(t types.Object) bool {
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

func (pi *PackageInfo) AddType(t types.Object) (err error) {
	// check for struct
	if isStruct(t) {
		var m *Message
		m, err = pi.addMessage(t)

		if (err == nil) && (m != nil) {
			pi.helper.OnMessage(pi, m)

			for _, f := range m.Fields {
				pi.helper.OnField(pi, f)
			}
		}

		// look for enumeration values
	} else if isConst(t) {
		var e *EnumValue
		e, err = pi.addEnum(t)
		if (err == nil) && (e != nil) {
			pi.helper.OnEnumValue(pi, e)
		}
	} else if isMap(t) {
		var m *Map
		m, err = pi.addMap(t)
		if (err == nil) && (m != nil) {
			pi.helper.OnMap(pi, m)
		}
	}

	return
}

func (pi *PackageInfo) addMessage(t types.Object) (result *Message, err error) {
	if pi.shouldAddMessage(t) {
		result = CreateMessage(t, t.Type().Underlying().(*types.Struct), pi.p)
		pi.Messages = append(pi.Messages, result)
	}

	return
}

func (pi *PackageInfo) addEnum(t types.Object) (result *EnumValue, err error) {
	if e := pi.GetEnumForType(t); e != nil {
		if result, err = e.AddValue(t.(*types.Const)); err != nil {
			err = fmt.Errorf("unable to add enum value '%s' to enum: %s", t.Name(), err)
		}
	}

	return
}

func (pi *PackageInfo) addMap(t types.Object) (result *Map, err error) {
	m, ok := t.Type().Underlying().(*types.Map)
	if !ok {
		return nil, fmt.Errorf("expecting *types.Map, not %t", t)
	}

	if _, ok = pi.Maps[t.Name()]; ok {
		return nil, nil
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

	return
}
