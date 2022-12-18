package storage

import (
	_ "modernc.org/sqlite"
)

// Fake DB...  just for testing
// wont do anything with the data.
type NullDB struct {
}

// NewNullDB creates new NullDB
func NewNullDB() (*NullDB, error) {
	db := NullDB{}
	return &db, nil
}

func (db *NullDB) Add(objectID string, propertyID string, data []byte) error {
	return nil
}

// Delete objectID/propertyID from table.
func (db *NullDB) Delete(objectID string, propertyID string) error {
	return nil
}

// Update an existing objectID/propertyID with new data.
// Given Add has become an upsert, this function can probably go.
func (db *NullDB) Update(objectID string, propertyID string, data []byte) error {
	return nil
}

// Import will take a map of property/data and store it as an object.
func (db *NullDB) Import(objectID string, properties map[string][]byte) (string, error) {
	panic("Not implemented")
	return "", nil
}

// Get returns an object (id + property/data map)
func (db *NullDB) Get(objectID string) (*Object, error) {
	object := Object{ObjectID: objectID}
	return &object, nil
}
