package main

import (
	"os"

	"clean_codex_token/internal/app"
)

func main() {
	code := app.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
