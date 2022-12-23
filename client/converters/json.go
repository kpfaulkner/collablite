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

	allProperties := processKey("", res)

	obj := client.NewObject(objectID, "JSON")
	var a any
	for k, v := range allProperties {

		switch v.Type {
		case gjson.String:
			a = v.String()
		case gjson.Number:
			a = v.Num
		case gjson.True:
			a = true
		case gjson.False:
			a = false
		default:
			panic("NO IDEA")
		}

		bytes, err := json.Marshal(a)
		if err != nil {
			log.Errorf("Error marshalling JSON: %v", err)
			return nil, err
		}
		obj.Properties[k] = bytes
	}

	return obj, nil
}

func processKey(keyPrefix string, parsedJson gjson.Result) map[string]gjson.Result {
	allKeys := make(map[string]gjson.Result)
	parsedJson.ForEach(func(k, v gjson.Result) bool {
		var newKey string

		if keyPrefix != "" {
			newKey = fmt.Sprintf("%s.%s", keyPrefix, k.String())
		} else {
			newKey = k.String()
		}

		if v.Type == gjson.JSON {
			res := gjson.Parse(v.Raw)
			m := processKey(newKey, res)
			for kk, vv := range m {
				allKeys[kk] = vv
			}
			return true
		} else {
			allKeys[newKey] = v
			return true
		}
		return true
	})

	return allKeys
}
