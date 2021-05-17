package main

import (
	"bytes"
	"encoding/gob"
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

func GetDBItem(key string, outPointer interface{}) error {
	var data []byte

	err := DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			// This func with val will only be called if item.Value encounters no error.
			data = append([]byte{}, val...)
			return nil
		})

		return err
	})

	if err != nil {
		log.Printf("Error getting item %s from badger: %v", key, err)
		return err
	}

	decoder := gob.NewDecoder(bytes.NewReader(data))

	return decoder.Decode(outPointer)
}

func encodeForDB(item interface{}) ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)

	err := e.Encode(item)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
