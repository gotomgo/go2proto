package main

import (
	"errors"
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"text/template"

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
		info := processPackage(p)

		if outputFileName, err := writePackageOutput(info, *targetFolder); err != nil {
			log.Fatalf("error writing output: %s", err)
		} else {
			log.Printf("output file written to '%s%s%s'\n", pwd, string(os.PathSeparator), outputFileName)
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
			errs += fmt.Sprintf("error fetching package '%s': ", p.String())
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

func processPackage(p *packages.Package) (result PackageInfo) {
	result = NewPackageInfo(p, NewProtoHelper())

	for _, t := range p.TypesInfo.Defs {
		if t == nil || !t.Exported() {
			continue
		}

		if err := result.AddType(t); err != nil {
			fmt.Printf("error adding type '%s': %s", t.Name(), err)
		}
	}

	// fixup the message and enum defintions
	result.Canonicalize()

	return
}

func writePackageOutput(info PackageInfo, path string) (outputFileName string, err error) {
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
		err = fmt.Errorf("unable to create file '%s': %s", outputFileName, err)
		return
	}
	defer f.Close()

	err = tmpl.Execute(f, info)

	return
}
