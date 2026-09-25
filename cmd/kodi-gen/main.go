// Command kodi-gen reads a Kodi JSON-RPC introspection schema directory
// and generates a versioned client package. All logic lives in
// internal/gen; this is a thin CLI wrapper.
//
// Usage:
//
//	go run ./cmd/kodi-gen -schema schema/v13 -out v13 -package v13
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ayasechan/kodigo/internal/gen"
)

func main() {
	schemaDir := flag.String("schema", "schema", "schema directory")
	outDir := flag.String("out", ".", "output directory for generated files")
	pkgName := flag.String("package", "", "Go package name for generated files (default: base of -out)")
	rpcImport := flag.String("rpcimport", "github.com/ayasechan/kodigo/internal/rpc", "import path of the shared RPC transport package (package name must be rpc)")
	flag.Parse()

	err := gen.Generate(gen.Config{
		SchemaDir: *schemaDir,
		OutDir:    *outDir,
		Package:   *pkgName,
		RPCImport: *rpcImport,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "kodi-gen:", err)
		os.Exit(1)
	}
}
