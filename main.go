package main

import (
	"log"
	"os"

	"github.com/BRAVO68WEB/fwdx/cmd/fwdx"
)

func main() {
	if err := fwdx.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
