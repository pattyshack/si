package main

import (
	"fmt"
	"io"
	"os"

	"github.com/pattyshack/gt/parseutil"

	lex "github.com/pattyshack/chickadee/parser/lexer"
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

		reader := parseutil.NewBufferedByteLocationReaderFromSlice(
			fileName,
			content)

		lexer := lex.NewLexer(reader)
		for {
			token, err := lexer.Next()
			if err != nil {
				if err != io.EOF {
					fmt.Println("Lex error:", err)
				}
				break
			}

			fmt.Println(token)
		}
	}
}
