package client

// OutgoingChange is the change the client is sending to the server.
// The change is purely related to a single property within the object.
// If there has been 2 changes (eg, colour changed to red and size changed to 10) then
// two separate OutgoingChange objects will need to be created and sent.
type OutgoingChange struct {
	ObjectID   string
	PropertyID string
	Data       []byte
}

// ChangeConfirmation is the confirmation that the server accepts this change and has passed to all
// subscribed clients
// Currently this has the same structure as OutgoingChange, but keeping them as separate types due to
// suspecting they will diverge
type ChangeConfirmation struct {
	ObjectID   string
	PropertyID string
	Data       []byte
}

// Object is a simple object used to represent an object in the system.
type Object struct {
	ObjectID   string
	Properties map[string][]byte

	// This may or may not stay. This is just to help the client know what type of
	// object this is representing. eg Could be set to "json" so they know they can
	// change to JSON later, or anything else. This is purely a helper field.
	ObjectType string
}

func NewObject(objectID string, objectType string) *Object {
	o := &Object{
		ObjectID:   objectID,
		ObjectType: objectType,
		Properties: make(map[string][]byte),
	}
	return o
}
