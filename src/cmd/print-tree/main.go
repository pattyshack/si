package main

import (
	"fmt"
	"os"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser"
	"github.com/pattyshack/chickadee/platform"
	"github.com/pattyshack/chickadee/platform/amd64"
)

func main() {
	targetPlatform := amd64.NewPlatform(platform.Linux)

	for _, fileName := range os.Args[1:] {
		fmt.Println("=====================")
		fmt.Println("File name:", fileName)
		fmt.Println("---------------------")
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Println("ReadFile error:", err)
			continue
		}

		emitter := &parseutil.Emitter{}
		entries := parser.Parse(
			parseutil.NewBufferedByteLocationReaderFromSlice(
				fileName,
				content),
			emitter)

		analyzer.Analyze(entries, targetPlatform, emitter)

		for idx, entry := range entries {
			fmt.Printf("Entry %d:\n", idx)
			fmt.Println(ast.TreeString(entry, "  "))
		}

		errs := emitter.Errors()
		if len(errs) > 0 {
			fmt.Println("---------------------------")
			fmt.Println("Found", len(errs), "errors:")
			fmt.Println("---------------------------")
			for idx, err := range errs {
				fmt.Printf("error %d: %s\n", idx, err)
			}
		}
	}
}
