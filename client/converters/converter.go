package converters

import (
	"github.com/kpfaulkner/collablite/client"
)

// Converter is the general interface clients will need to use to convert from whatever their
// structure is to our internal Object struct.
type Converter interface {

	// Converts from our internal object to what the client wants.
	// Returns objectid, actual object and error.
	// Keeping objectid separate due to not knowing if the client object has an objectid property.
	ConvertFromObject(object client.Object) (string, any, error)

	// ConvertToObject converts a clients object TO the internal object.
	// It takes in an existing internal object (if one exists) and updates it with the new data.
	// It returns a pointer to the internal object.
	// If the existingObject is nil, then it creates a new one.
	ConvertToObject(objectID string, exitingObject *client.Object, clientObject any) (*client.Object, error)
}
