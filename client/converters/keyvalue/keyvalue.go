package keyvalue

import (
	"errors"

	"github.com/kpfaulkner/collablite/client"
)

// KeyValueObject is just a default structure for the client. Basic map of strings to byte arrays
type KeyValueObject struct {
	ObjectID   string
	Properties map[string][]byte
}

// ConvertFromObject converts an object to KEYVALUE representation
// Doesn't really do any conversion...  this is just a default converter where
// its basically the same as the underlying object.
func ConvertFromObject(object client.Object) (string, any, error) {

	if object.ObjectType == "KEYVALUE" {
		kv := KeyValueObject{
			ObjectID:   object.ObjectID,
			Properties: make(map[string][]byte),
		}

		for k, v := range object.Properties {
			kv.Properties[k] = v.Data
		}
		return object.ObjectID, &kv, nil
	}
	return "", nil, errors.New("Not KeyValue")
}

func ConvertToObject(objectID string, exitingObject *client.Object, clientObject any) (*client.Object, error) {

	var obj *client.Object
	if exitingObject == nil {
		obj = client.NewObject(objectID, "KEYVALUE")
	} else {
		obj = exitingObject
	}

	keyValueObject := clientObject.(KeyValueObject)
	for k, v := range keyValueObject.Properties {
		obj.AdjustProperty(k, v)
	}
	return obj, nil
}
