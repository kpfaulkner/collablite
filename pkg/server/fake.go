package server

import (
	"errors"

	"github.com/kpfaulkner/collablite/pkg/storage"
)

type fakeDB struct {
	data map[string]map[string][]byte
}

// NewFakeDB creates new NullDB
func NewFakeDB() (*fakeDB, error) {
	db := fakeDB{}
	db.data = make(map[string]map[string][]byte)
	return &db, nil
}

func (db *fakeDB) Add(objectID string, propertyID string, data []byte) error {
	if d, ok := db.data[objectID]; !ok {
		db.data[objectID] = make(map[string][]byte)
		db.data[objectID][propertyID] = data
	} else {
		d[propertyID] = data
	}

	return nil
}

// Delete objectID/propertyID from table.
func (db *fakeDB) Delete(objectID string, propertyID string) error {
	return nil
}

// Update an existing objectID/propertyID with new data.
// Given Add has become an upsert, this function can probably go.
func (db *fakeDB) Update(objectID string, propertyID string, data []byte) error {
	return nil
}

// Import will take a map of property/data and store it as an object.
func (db *fakeDB) Import(objectID string, properties map[string][]byte) (string, error) {
	panic("Not implemented")
	return "", nil
}

// Get returns an object (id + property/data map)
func (db *fakeDB) Get(objectID string) (*storage.Object, error) {
	if obj, ok := db.data[objectID]; ok {
		object := storage.NewObject(objectID)
		for prop, val := range obj {
			object.Properties[prop] = val
		}
		return object, nil
	}
	return nil, errors.New("no object")
}
