package main

import (
	"log"
	"os"
)

func main() {
	// Initialize your Postgres storage
	store, err := NewPostgresStorage()
	if err != nil {
		log.Fatal(err)
	}
	store.Init()

	// Set up the static directory path
	// staticDir := "public/"

	// Set up server configuration using environment variables
	// dbHost := os.Getenv("DB_HOST")
	// dbPort := os.Getenv("DB_PORT")
	// dbUser := os.Getenv("DB_USER")
	// dbPassword := os.Getenv("DB_PASSWORD")
	// dbName := os.Getenv("DB_NAME")
	server := NewAPIServer(":"+os.Getenv("PORT"), store, staticDir, dbHost, dbPort, dbUser, dbPassword, dbName)

	// Run the server
	server.Run()
}
