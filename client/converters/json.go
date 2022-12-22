package converters

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kpfaulkner/collablite/client"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ObjectToJSON converts an object to JSON representation
// Currently it is really up to the caller to know if they're really dealing with JSON
// or not. If this is called and the object is not JSON, there is no guarantee what will result.
// The object has a "hint" of the type, but this is not enforced.
func ObjectToJson(object client.Object) (string, error) {

	if object.ObjectType == "JSON" {

		var newJson string
		for k, v := range object.Properties {
			var a any
			err := json.Unmarshal(v, &a)
			if err != nil {
				log.Errorf("Error unmarshalling JSON: %v", err)
				return "", err
			}
			newJson, _ = sjson.Set(newJson, k, a)
		}
		return newJson, nil
	}

	return "", errors.New("Not JSON")
}

func JsonToObject(objectID string, j string) (*client.Object, error) {
	res := gjson.Parse(j)

	allKeys := make(map[string]any)
	res.ForEach(func(key, value gjson.Result) bool {
		fmt.Println("key", key, "value", value)
		keys := processKey(key.String(), value, j)
		for k, v := range keys {
			allKeys[k] = v
		}
		return true
	})

	obj := client.Object{}
	obj.ObjectType = "JSON"
	obj.ObjectID = objectID
	for k, v := range allKeys {
		bytes, err := json.Marshal(v)
		if err != nil {
			log.Errorf("Error marshalling JSON: %v", err)
			return nil, err
		}
		obj.Properties[k] = bytes
	}

	return nil, errors.New("Not JSON")
}

func processKey(key string, parsedJson gjson.Result, origJson string) map[string]any {

	allKeys := make(map[string]any)
	parsedJson.ForEach(func(k, v gjson.Result) bool {

		var newKey string
		if k.String() == "" {

			switch v.Type {
			case gjson.String:
				allKeys[key] = v.Str
			case gjson.Number:
				allKeys[key] = v.Num
			case gjson.JSON:
				// dont want to do anything... we'll just keep looping
			default:
				panic("NO IDEA")
			}

			return true
		} else {
			newKey = fmt.Sprintf("%s.%s", key, k)
		}
		res := gjson.Get(origJson, newKey)
		keys := processKey(newKey, res, origJson)
		for k, v := range keys {
			allKeys[k] = v
		}
		return true
	})

	return allKeys
}
