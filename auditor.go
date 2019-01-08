package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/lukaszbudnik/auditor/server"
	"github.com/lukaszbudnik/auditor/store/provider"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Could not load .env file")
	}
	store, err := provider.NewStore()
	if err != nil {
		log.Fatalf("Could not connect to backedn store: %v", err.Error())
	}
	_, err = server.Start(store)
	if err != nil {
		log.Fatalf("Could not start server: %v", err.Error())
	}
}
