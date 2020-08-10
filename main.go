package main

import (
	"errors"
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structtag"
	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/packages"
)

type arrFlags []string

func (i *arrFlags) String() string {
	return ""
}

func (i *arrFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	filter       = flag.String("filter", "", "Filter by struct names. Case insensitive.")
	targetFolder = flag.String("f", ".", "Protobuf output file path.")
	pkgFlags     arrFlags
)

func main() {
	flag.Var(&pkgFlags, "p", `Fully qualified path of packages to analyse. Relative paths ("./example/in") are allowed.`)
	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting working directory: %s", err)
	}

	if len(pkgFlags) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	//ensure the path exists
	_, err = os.Stat(*targetFolder)
	if err != nil {
		log.Fatalf("error getting output file: %s", err)
	}

	pkgs, err := loadPackages(pwd, pkgFlags)
	if err != nil {
		log.Fatalf("error fetching packages: %s", err)
	}

	for _, p := range pkgs {
		info := getMessagesForPackage(p)

		if outputFileName, err := writePackageOutput(info, *targetFolder); err != nil {
			log.Fatalf("error writing output: %s", err)
		} else {
			log.Printf("output file written to %s%s%s\n", pwd, string(os.PathSeparator), outputFileName)
		}
	}
}

// attempt to load all packages
func loadPackages(pwd string, pkgs []string) ([]*packages.Package, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Dir:  pwd,
		Mode: packages.LoadSyntax,
		Fset: fset,
	}
	packages, err := packages.Load(cfg, pkgs...)
	if err != nil {
		return nil, err
	}
	var errs = ""
	//check each loaded package for errors during loading
	for _, p := range packages {
		if len(p.Errors) > 0 {
			errs += fmt.Sprintf("error fetching package %s: ", p.String())
			for _, e := range p.Errors {
				errs += e.Error()
			}
			errs += "; "
		}
	}
	if errs != "" {
		return nil, errors.New(errs)
	}
	return packages, nil
}

type packageInfo struct {
	p    *packages.Package
	seen map[string]bool

	Name     string
	Path     string
	Messages []*message
	Imports  []string
	Enums    map[string]*enum
}

func newPackageInfo(p *packages.Package) packageInfo {
	return packageInfo{
		p:     p,
		Name:  p.Name,
		Path:  p.PkgPath,
		seen:  map[string]bool{},
		Enums: map[string]*enum{},
	}
}

type message struct {
	Name   string
	Fields []*field
}

type field struct {
	Name       string
	TypeName   string
	Order      int
	IsRepeated bool
	JSONName   string
}

type enum struct {
	Name           string
	Values         []*enumValue
	AllowAlias     bool
	MissingDefault bool
}

type enumValue struct {
	Name  string
	Value int64
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

func getMessagesForPackage(p *packages.Package) (result packageInfo) {
	result = newPackageInfo(p)

	for _, t := range p.TypesInfo.Defs {
		if t == nil {
			continue
		}
		if !t.Exported() {
			continue
		}

		if s, ok := t.Type().Underlying().(*types.Struct); ok {
			if _, ok := result.seen[t.Type().String()]; ok {
				continue
			}

			result.seen[t.Type().String()] = true

			typePkgName := getPackageFromType(t.Type().String())
			if typePkgName != t.Pkg().Path() {
				result.Imports = append(result.Imports, typePkgName)
				continue
			}

			result.Messages = appendMessage(result.Messages, t, s, p)

			// look for enumeration values
		} else if c, ok := t.(*types.Const); ok {
			// should be of type int (or int64?)
			if t.Type().Underlying().String() == "int" {
				// the enum type for the const must be defined in the same package
				if getPackageFromType(t.Type().String()) == result.p.PkgPath {
					// enum Type Name
					enumTypeName := splitTypeNameHelper(t.Type())

					// have we already seen it?
					e, ok := result.Enums[enumTypeName]

					// If not, create the enum type and remember it
					if !ok {
						e = &enum{Name: enumTypeName}
						result.Enums[enumTypeName] = e
					}

					// convert the enum value to int64
					val, err := strconv.ParseInt(c.Val().String(), 10, 64)
					if err == nil {
						// add the value to the enum type
						e.Values = append(e.Values, &enumValue{Name: strcase.ToScreamingSnake(c.Name()), Value: val})
					} else {
						fmt.Printf("error: unable to convert const value '%s' to type int64: %s\n", c.Val(), err)
					}
				}
			}
		}
	}

	// sort messages (structs) by name
	sort.Slice(result.Messages, func(i, j int) bool { return result.Messages[i].Name < result.Messages[j].Name })

	for _, enum := range result.Enums {
		// sort enum values by value
		sort.Slice(enum.Values, func(i, j int) bool {
			if enum.Values[i].Value == enum.Values[j].Value {
				return enum.Values[i].Name < enum.Values[j].Name
			}
			return enum.Values[i].Value < enum.Values[j].Value
		})

		hasZero := false
		hasAlias := false
		var lastValue int64
		var pLastValue *int64

		for _, val := range enum.Values {
			if val.Value == 0 {
				hasZero = true
			}

			if (pLastValue != nil) && (val.Value == *pLastValue) {
				hasAlias = true
			}

			lastValue = val.Value
			pLastValue = &lastValue
		}

		enum.MissingDefault = !hasZero
		enum.AllowAlias = hasAlias
	}

	spew.Dump(result.Enums)
	return
}

func appendMessage(out []*message, t types.Object, s *types.Struct, p *packages.Package) []*message {
	msg := &message{
		Name:   t.Name(),
		Fields: []*field{},
	}

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

		newField := &field{
			Name:       toProtoFieldName(f.Name()),
			TypeName:   toProtoFieldTypeName(f, p),
			IsRepeated: isRepeated(f),
			Order:      i + 1,
			JSONName:   jsonName,
		}
		msg.Fields = append(msg.Fields, newField)
	}
	out = append(out, msg)
	return out
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

func splitNameHelper(f *types.Var) string {
	return splitTypeNameHelper(f.Type())
}

func normalizeType(name string, p *packages.Package) (result string) {
	switch name {
	case "int":
		result = "int64"
	case "float32":
		result = "float"
	case "float64":
		result = "double"
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

func isRepeated(f *types.Var) bool {
	_, ok := f.Type().Underlying().(*types.Slice)
	return ok
}

func toProtoFieldName(name string) string {
	if len(name) == 2 {
		return strings.ToLower(name)
	}

	// r, n := utf8.DecodeRuneInString(name)
	// return string(unicode.ToLower(r)) + name[n:]

	return strcase.ToSnake(name)
}

func writePackageOutput(info packageInfo, path string) (outputFileName string, err error) {
	msgTemplate := `syntax = "proto3";
package {{.Name}};

option go_package = "proto/{{.Path}}";

{{- range .Imports}}
import "{{.}}.proto";
{{- end}}
{{- range .Enums}}

enum {{.Name}} {
	{{- if .AllowAlias}}
	option allow_alias = true;{{- end}}
	{{- if .MissingDefault}}
	UNKOWN = 0;{{- end}}
	{{- range .Values}}
	{{.Name}} = {{.Value}};
	{{- end}}
}
{{- end}}

{{range .Messages}}
message {{.Name}} {
{{- range .Fields}}
{{- if .IsRepeated}}
  repeated {{.TypeName}} {{.Name}} = {{.Order}};
{{- else}}
  {{.TypeName}} {{.Name}} = {{.Order}} {{- if .JSONName}} [json_name="{{.JSONName}}"] {{- end}};
{{- end}}
{{- end}}
}
{{end}}
`
	tmpl, err := template.New("test").Parse(msgTemplate)
	if err != nil {
		panic(err)
	}

	outputFileName = fmt.Sprintf("%s.proto", info.Name)

	f, err := os.Create(filepath.Join(path, outputFileName))
	if err != nil {
		err = fmt.Errorf("unable to create file %s : %s", outputFileName, err)
		return
	}
	defer f.Close()

	err = tmpl.Execute(f, info)

	return
}
