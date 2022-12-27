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

func NewKeyValueObject(objectID string) *KeyValueObject {
	kv := KeyValueObject{
		ObjectID:   objectID,
		Properties: make(map[string][]byte),
	}
	return &kv
}

// ConvertFromObject converts an object to KEYVALUE representation
// Doesn't really do any conversion...  this is just a default converter where
// its basically the same as the underlying object.
func (kv *KeyValueObject) ConvertFromObject(object client.InternalObject) error {

	if object.ObjectType == "KEYVALUE" {

		for k, v := range object.Properties {
			kv.Properties[k] = v.Data
		}
		return nil
	}
	return errors.New("Not KeyValue")
}

func (kv *KeyValueObject) ConvertToObject(objectID string, exitingObject *client.InternalObject, clientObject any) (*client.InternalObject, error) {

	var obj *client.InternalObject
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
