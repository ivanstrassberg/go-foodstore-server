package main

import (
	"log"
	"os"
)


func main() {

	store, err := NewPostgresStorage()
	store.Init()
	if err != nil {
		log.Fatal(err)
	}
	// use to seed DB with data once!
	// store.SeedWithData("/Users/ivansilin/Documents/coding/golang/foodShop/rewritten/draft.txt")
	staticDir := "/Users/ivansilin/Documents/coding/golang/foodShop/initHandle/static/"
	server := NewAPIServer(":"+os.Getenv("PORT"), store, staticDir)

	server.Run()

}
