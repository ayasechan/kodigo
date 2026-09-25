package gen

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config controls one generator run.
type Config struct {
	// SchemaDir holds types.json, methods.json, notifications.json, version.txt.
	SchemaDir string
	// OutDir receives types.go, methods.go, notifications.go, client.go.
	OutDir string
	// Package is the Go package name of emitted files.
	// Default: base of OutDir ("kodi" for ".").
	Package string
	// RPCImport is the import path of the shared transport package.
	// Its package name must be rpc. Default: "github.com/ayasechan/kodigo/internal/rpc".
	RPCImport string
}

// Generate runs the generator once.
func Generate(cfg Config) error {
	pkg := cfg.Package
	if pkg == "" {
		pkg = filepath.Base(cfg.OutDir)
		if pkg == "." || pkg == string(filepath.Separator) {
			pkg = "kodi"
		}
	}
	rpcImport := cfg.RPCImport
	if rpcImport == "" {
		rpcImport = "github.com/ayasechan/kodigo/internal/rpc"
	}
	g := &generator{
		pkg:        pkg,
		rpcImport:  rpcImport,
		schemaDir:  cfg.SchemaDir,
		knownKinds: map[string]string{},
		defined:    map[string]bool{},
		missing:    map[string]bool{},
		usedConsts: map[string]bool{},
	}
	return g.run(cfg.SchemaDir, cfg.OutDir)
}

type generator struct {
	pkg       string // Go package name for generated files
	rpcImport string // import path of the shared transport package
	schemaDir string // schema directory, used in file headers

	typeOrder  []string
	typeRaw    map[string]json.RawMessage
	methodKeys []string
	methodRaw  map[string]json.RawMessage
	notifKeys  []string
	notifRaw   map[string]json.RawMessage
	version    string

	knownKinds map[string]string // sanitized type name -> kind: struct,slice,string,int,float,bool,any,map
	defined    map[string]bool
	missing    map[string]bool
	usedConsts map[string]bool

	// pending named definitions created while generating (methods/notifications)
	extraDefs []string
}

func (g *generator) run(schemaDir, outDir string) error {
	var err error
	read := func(name string) []byte {
		b, e := os.ReadFile(filepath.Join(schemaDir, name))
		if e != nil {
			err = e
		}
		return b
	}
	typesData := read("types.json")
	methodsData := read("methods.json")
	notifsData := read("notifications.json")
	verData := read("version.txt")
	if err != nil {
		return err
	}
	g.version = strings.TrimSpace(strings.TrimPrefix(string(verData), "JSONRPC_VERSION"))
	g.version = strings.Fields(g.version)[len(strings.Fields(g.version))-1]
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	if g.typeOrder, g.typeRaw, err = orderedObject(typesData); err != nil {
		return fmt.Errorf("parse types.json: %w", err)
	}
	if g.methodKeys, g.methodRaw, err = orderedObject(methodsData); err != nil {
		return fmt.Errorf("parse methods.json: %w", err)
	}
	if g.notifKeys, g.notifRaw, err = orderedObject(notifsData); err != nil {
		return fmt.Errorf("parse notifications.json: %w", err)
	}

	typesSrc, err := g.genTypes()
	if err != nil {
		return err
	}
	methodsSrc, err := g.genMethods()
	if err != nil {
		return err
	}
	notifsSrc, err := g.genNotifications()
	if err != nil {
		return err
	}
	clientSrc, err := g.genClient()
	if err != nil {
		return err
	}
	write := func(name, src string) error {
		formatted, err := format.Source([]byte(src))
		if err != nil {
			return fmt.Errorf("format %s: %w", name, err)
		}
		return os.WriteFile(filepath.Join(outDir, name), formatted, 0o644)
	}
	if err := write("types.go", typesSrc); err != nil {
		return err
	}
	if err := write("methods.go", methodsSrc); err != nil {
		return err
	}
	if err := write("notifications.go", notifsSrc); err != nil {
		return err
	}
	if err := write("client.go", clientSrc); err != nil {
		return err
	}
	fmt.Printf("kodi-gen: schema version %s: %d types, %d methods, %d notifications\n",
		g.version, len(g.typeOrder), len(g.methodKeys), len(g.notifKeys))
	if len(g.missing) > 0 {
		names := make([]string, 0, len(g.missing))
		for n := range g.missing {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Printf("kodi-gen: %d unresolved $refs emitted as string stubs: %s\n",
			len(names), strings.Join(names, ", "))
	}
	return nil
}
