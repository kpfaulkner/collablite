package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {

	keyStr := flag.String("key", "", "key")
	flag.Parse()

	db, err := pebble.Open(`C:/Users/kenfa/projects/collablite/cmd/server/pebbledb`, &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}

	key := []byte(*keyStr)
	value, closer, err := db.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %s\n", key, value)
	if err := closer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}

}
