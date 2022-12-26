package storage

// Object represents an object in the system.
// Its VERY basic.
// ObjectID (unique identifier)
// Map of propertyname to bytes.
// The caller can make the property names whatever they want...   the server does not care
// (well shouldn't care...  TBD)
type Object struct {
	ObjectID   string
	Properties map[string][]byte
}

func NewObject(objectID string) *Object {
	o := &Object{
		ObjectID:   objectID,
		Properties: make(map[string][]byte),
	}
	return o
}

// DB interface used to store the data *somewhere*
type DB interface {

	// Add an object to the DB.
	//
	Add(objectID string, propertyID string, data []byte) error
	Delete(objectID string, propertyID string) error
	Update(objectID string, propertyID string, data []byte) error

	// Import imports an entire object (basically objectID and property collection)
	Import(objectID string, properties map[string][]byte) (string, error)

	// Get entire object. Will return all properties for an object.
	Get(objectID string) (*Object, error)
}
