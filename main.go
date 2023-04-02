package main

import (
	"fmt"
	"os"

	"github.com/nepomuceno/az-spec-validator/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "generate-docs" {
		fmt.Println("Generating docs...")
		doc.GenMarkdownTree(cmd.GetRootCmd(), "./docs")
	} else {
		cmd.Execute()
	}
}
