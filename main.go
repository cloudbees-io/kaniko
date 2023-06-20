package main

import (
	"log"

	"github.com/cloudbees-io/kaniko/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
