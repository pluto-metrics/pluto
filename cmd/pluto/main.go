package main

import (
	"flag"
	"log"

	"github.com/pluto-metrics/pluto/cmd/pluto/config"
)

func main() {
	var configFilename string
	flag.StringVar(&configFilename, "config", "config.yaml", "Config filename")
	flag.Parse()

	cfg, err := config.LoadFromFile(configFilename)
	if err != nil {
		log.Fatal(err)
	}

}
