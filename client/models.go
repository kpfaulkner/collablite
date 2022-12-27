package client

import "bytes"

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

// ClientObject is a simple object used to represent an object in the system.
type ClientObject struct {
	ObjectID   string
	Properties map[string]Property

	// This may or may not stay. This is just to help the client know what type of
	// object this is representing. eg Could be set to "json" so they know they can
	// change to JSON later, or anything else. This is purely a helper field.
	ObjectType string
}

// Property is the a property key/value of an object.
// We dont care what the data is, we purely treat it as a byte array
// Keep a dirty flag to know what data needs to be sent to server
type Property struct {
	Data []byte

	// Dirty is used to identify any property that has been changed locally and needs to be sent to the server.
	Dirty bool

	// indicates that its been updated from the server... and needs to be used by the client.
	Updated bool
}

func NewObject(objectID string, objectType string) *ClientObject {
	o := &ClientObject{
		ObjectID:   objectID,
		ObjectType: objectType,
		Properties: make(map[string]Property),
	}
	return o
}

func (o *ClientObject) AdjustProperty(propertyID string, data []byte) {
	if p, ok := o.Properties[propertyID]; ok {

		// check if data has changed.
		if bytes.Compare(p.Data, data) != 0 {
			p.Data = data
			p.Dirty = true
			o.Properties[propertyID] = p
		}
	} else {
		o.Properties[propertyID] = Property{Data: data, Dirty: true}
	}
}
