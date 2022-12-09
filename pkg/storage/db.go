package storage

// DB interface used to store the data *somewhere*
type DB interface {

	// Add an object to the DB.
	//
	Add(objectID string, propertyID string, data []byte) error
	Delete(objectID string, propertyID string) error
	Update(objectID string, propertyID string, data []byte) error

	// Import imports an entire object (basically objectID and property collection)
	Import(objectID string, properties map[string][]byte) (string, error)
}
