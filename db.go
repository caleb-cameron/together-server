package main

import (
	"log"

	badger "github.com/dgraph-io/badger/v3"
)

var DB *badger.DB

func initDB() {
	db, err := badger.Open(badger.DefaultOptions(config.BadgerDir))

	if err != nil {
		log.Fatal(err)
	}

	DB = db
}

func closeDB() {
	DB.Close()
}
