package main

import (
	"log"

	"github.com/external-secrets/bitwarden-sdk-server/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
