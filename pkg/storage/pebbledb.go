package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/pebble"
	log "github.com/sirupsen/logrus"
)

// PebbleMinimal is the minimal interface we use from Pebble
// Interface to help mock this out for testing.
type PebbleMinimal interface {
	Set(key, value []byte, opts *pebble.WriteOptions) error
	NewIter(o *pebble.IterOptions) *pebble.Iterator
}

// BadgerDB implements the DB interface using BadgerDB
type PebbleDB struct {
	pdb PebbleMinimal
	ctx context.Context
}

func NewPebbleClient(dir string) (*pebble.DB, error) {
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

// NewPebbleDB creates new BadgerDB DB connection
func NewPebbleDB(pdb PebbleMinimal) (*PebbleDB, error) {
	dbs := PebbleDB{}
	dbs.pdb = pdb
	dbs.ctx = context.Background()
	return &dbs, nil
}

func (db *PebbleDB) Add(objectID string, propertyID string, data []byte) error {
	key := fmt.Sprintf("%s:%s", objectID, propertyID)
	if err := db.pdb.Set([]byte(key), data, pebble.Sync); err != nil {
		log.Fatal(err)
	}

	return nil
}

// Delete objectID/propertyID from table.
func (db *PebbleDB) Delete(objectID string, propertyID string) error {
	panic("Not implemented")
	return nil
}

// Update an existing objectID/propertyID with new data.
// Given Add has become an upsert, this function can probably go.
func (db *PebbleDB) Update(objectID string, propertyID string, data []byte) error {

	// just do Add...
	db.Add(objectID, propertyID, data)
	return nil
}

// Import will take a map of property/data and store it as an object.
func (db *PebbleDB) Import(objectID string, properties map[string][]byte) (string, error) {
	panic("Not implemented")
	return "", nil
}

// Get returns an object (id + property/data map)
func (db *PebbleDB) Get(objectID string) (*Object, error) {

	objectProperties := make(map[string][]byte)

	iter := db.pdb.NewIter(&pebble.IterOptions{
		LowerBound: []byte(objectID),
	})

	for iter.First(); iter.Valid(); iter.Next() {
		// Only keys beginning with "prefix" will be visited.
		key := iter.Key()
		v, err := iter.ValueAndErr()
		if err != nil {
			log.Fatal(err)
		}
		sp := strings.Split(string(key), ":")
		objectProperties[sp[1]] = v
	}

	object := Object{ObjectID: objectID, Properties: objectProperties}
	return &object, nil
}
