package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/lukaszbudnik/auditor/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Could not load .env file")
	}
	server.Start()
}
