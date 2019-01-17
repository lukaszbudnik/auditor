package main

import (
	"flag"
	"log"

	"github.com/joho/godotenv"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/server"
	"github.com/lukaszbudnik/auditor/store/provider"
)

const (
	// DefaultConfigFile defines default file name of migrator configuration file
	DefaultConfigFile = ".env"
)

func main() {
	// fail fast, ValidateBlockType method panics if type of passed struct is invalid
	// this simplifies operations in the rest of the auditor code
	model.ValidateBlockType(&model.Block{})

	var configFile string
	flag.StringVar(&configFile, "configFile", "", "optional argument with a name of configuration file to use")
	flag.Parse()
	if len(configFile) == 0 {
		configFile = DefaultConfigFile
	}
	if err := godotenv.Load(configFile); err != nil {
		log.Fatalf("FATAL Could not load configuration file: %v", err.Error())
	}
	log.Printf("INFO auditor read configuration from file: %v", configFile)
	store, err := provider.NewStore()
	if err != nil {
		log.Fatalf("FATAL Could not connect to backend store: %v", err.Error())
	}
	_, err = server.Start(store)
	if err != nil {
		log.Fatalf("FATAL Could not start server: %v", err.Error())
	}
}
