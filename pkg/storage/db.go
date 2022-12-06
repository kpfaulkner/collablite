package storage

// DB interface used to store the data *somewhere*
// Expectation is that the data is in a JSON(-ish) format.
// ObjectID is just some unique identifier
// Path is a JSON path... e.g. /foo/bar/baz
// Data will be byte array verson of JSON. Maybe switch this to a string?
type DB interface {

	// Add an object to the DB.
	//
	Add(objectID string, path string, data []byte) error
	Delete(objectID string, path string) error
	Update(objectID string, path string, data []byte) error

	// Import an entire JSON structure to the DB
	// return objectID
	Import(data []byte) (string, error)
}
