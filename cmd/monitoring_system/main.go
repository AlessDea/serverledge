package main

import (
	"log"

	"github.com/grussorusso/serverledge/ms"
)

func main() {
	// start the monitoryn system engine
	ms.Init()
	log.Printf("Started the Monitoring System engine\n")

}
