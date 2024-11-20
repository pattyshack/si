package main

import (
	"fmt"
	"os"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser"
)

func main() {
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
