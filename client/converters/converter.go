package converters

import (
	"github.com/kpfaulkner/collablite/client"
)

// Converter is the general interface clients will need to use to convert from whatever their
// structure is to our internal InternalObject struct.
type Converter interface {

	// Converts from our internal object to what the client wants.
	// It is up to the client to decide what to do with the data.
	// Instead of just passing the specific change we pass the updated internal object.
	// This might be a waste since the client callback will need to determine what has changed.
	// Possible revisit this.  TODO(kpfaulkner)
	ConvertFromObject(object client.InternalObject) error

	// ConvertToObject converts a clients object TO the internal object.
	// It takes in an existing internal object (if one exists) and updates it with the new data.
	// It returns a pointer to the internal object.
	// If the existingObject is nil, then it creates a new one.
	ConvertToObject(objectID string, exitingObject *client.InternalObject, clientObject any) (*client.InternalObject, error)
}
