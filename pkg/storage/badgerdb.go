package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	log "github.com/sirupsen/logrus"
)

// BadgerDB implements the DB interface using BadgerDB
type BadgerDB struct {
	bdb *badger.DB
	ctx context.Context
}

// NewBadgerDB creates new BadgerDB DB connection
func NewBadgerDB(dir string) (*BadgerDB, error) {
	dbs := BadgerDB{}
	db, err := badger.Open(badger.DefaultOptions(dir))
	if err != nil {
		log.Fatal(err)
	}
	dbs.bdb = db
	dbs.ctx = context.Background()
	go dbs.startGC()
	return &dbs, nil
}

func (db *BadgerDB) startGC() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		log.Debug("GC ticker")
	again:
		err := db.bdb.RunValueLogGC(0.7)
		if err == nil {
			goto again
		}
	}

	log.Debugf("GC completed/failed")
}

func (db *BadgerDB) Add(objectID string, propertyID string, data []byte) error {
	key := fmt.Sprintf("%s:%s", objectID, propertyID)
	err := db.bdb.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), data)
		return err
	})
	if err != nil {
		log.Errorf("unable to add object %v", err)
		return err
	}

	return nil
}

// Delete objectID/propertyID from table.
func (db *BadgerDB) Delete(objectID string, propertyID string) error {
	panic("Not implemented")
	return nil
}

// Update an existing objectID/propertyID with new data.
// Given Add has become an upsert, this function can probably go.
func (db *BadgerDB) Update(objectID string, propertyID string, data []byte) error {

	// just do Add...
	db.Add(objectID, propertyID, data)
	return nil
}

// Import will take a map of property/data and store it as an object.
func (db *BadgerDB) Import(objectID string, properties map[string][]byte) (string, error) {
	panic("Not implemented")
	return "", nil
}

// Get returns an object (id + property/data map)
func (db *BadgerDB) Get(objectID string) (*Object, error) {

	objectProperties := make(map[string][]byte)

	db.bdb.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(objectID)
		var valCopy []byte
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				sp := strings.Split(string(k), ":")
				valCopy = append([]byte{}, v...)
				objectProperties[sp[1]] = valCopy
				return nil
			})
			if err != nil {
				log.Errorf("unable to get object %v", err)
				return err
			}
		}
		return nil
	})

	object := Object{ObjectID: objectID, Properties: objectProperties}
	return &object, nil
}
