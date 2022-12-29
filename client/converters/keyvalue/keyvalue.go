package keyvalue

import (
	"sync"

	"github.com/kpfaulkner/collablite/client"
	log "github.com/sirupsen/logrus"
)

// KeyValueObject is just a default structure for the client. Basic map of strings to byte arrays
type KeyValueObject struct {
	ObjectID   string
	Properties map[string][]byte

	Lock sync.Mutex
}

func NewKeyValueObject(objectID string) *KeyValueObject {
	kv := KeyValueObject{
		ObjectID:   objectID,
		Properties: make(map[string][]byte),
	}
	return &kv
}

func (kv *KeyValueObject) GetProperties() map[string][]byte {
	kv.Lock.Lock()
	defer kv.Lock.Unlock()
	newMap := make(map[string][]byte, len(kv.Properties))
	for k, v := range kv.Properties {
		newMap[k] = v
	}
	return newMap
}

// ConvertFromObject converts an object to KEYVALUE representation
// Doesn't really do any conversion...  this is just a default converter where
// its basically the same as the underlying object.
func (kv *KeyValueObject) ConvertFromObject(object *client.ClientObject) error {

	kv.Lock.Lock()

	properties := object.GetProperties()
	for k, v := range properties {
		if v.Updated {
			kv.Properties[k] = v.Data
			log.Debugf("Got update for property %s : %s", k, string(v.Data))
		}
	}
	kv.Lock.Unlock()

	return nil
}

func (kv *KeyValueObject) ConvertToObject(objectID string, exitingObject *client.ClientObject, clientObject any) (*client.ClientObject, error) {

	var obj *client.ClientObject
	if exitingObject == nil {
		obj = client.NewObject(objectID, "KEYVALUE")
	} else {
		obj = exitingObject
	}

	keyValueObject := clientObject.(*KeyValueObject)

	properties := keyValueObject.GetProperties()

	for k, v := range properties {
		obj.AdjustProperty(k, v, true, false)
	}
	return obj, nil
}
