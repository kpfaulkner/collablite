package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kpfaulkner/collablite/client"
	"github.com/kpfaulkner/collablite/client/converters"
	"github.com/kpfaulkner/collablite/cmd/common"
	log "github.com/sirupsen/logrus"
)

const (
	jsonTemplate = `{ 
						"displayFieldName" : "<displayFieldName>",
						"fieldAliases" : {
						  "fieldName1" : "<fieldAlias1>",
						  "fieldName2" : "<fieldAlias2>"
						},
						"geometryType" : "<geometryType>",
						"hasZ" : true,  
						"hasM" : false,   
						"spatialReference" : "spatialReference"",
						"fields": [
									{
										"name": "field1",
										"type": "field1Type",
										"alias": "field1Alias"
									},
									{
										"name": "field2",
										"type": "field2Type",
										"alias": "field2Alias"
									}
								],
						 "features": [
									{
										"geometry": {
											"geo1"
										},
										"attributes": {
											"field1": 123,
											"field2": 234 
										} 
									},
									{
										"geometry": {
											"geo2"
										},
										"attributes": {
											"field1": 345,
											"field2": 456 
										} 
									}
								]
						}`
)

var obj *client.Object

func processObjectConfirmation(change *client.ChangeConfirmation) error {
	log.Debugf("confirmation: %v", change)

	// if propertyID empty (need to confirm why!!!) then skip it.
	if change.PropertyID != "" {
		obj.Properties[change.PropertyID] = change.Data
		j, err := converters.ObjectToJson(*obj)
		if err != nil {
			log.Errorf("error converting object to json: %v", err)
			return err
		}
	}
	return nil
}

func main() {
	fmt.Printf("So it begins...\n")
	host := flag.String("host", "localhost:50051", "host:port of server")
	objectID := flag.String("objectid", "testobject1", "objectid of object to write/watch")
	send := flag.Bool("send", false, "send data to server")
	logLevel := flag.String("loglevel", "info", "Log Level: debug, info, warn, error")
	flag.Parse()

	common.SetLogLevel(*logLevel)

	cli := client.NewClient(*host)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx := context.Background()
	cli.RegisterCallback(processObjectConfirmation)
	cli.Connect(ctx)
	go cli.Listen(ctx)

	var err error
	if *send {

		obj, err = converters.JsonToObject(*objectID, jsonTemplate)
		if err != nil {
			log.Errorf("error converting json to object: %v", err)
			return
		}

		go func() {
			for i := 0; i < 1000000000; i++ {

				req, err := generateRandomChange(obj)
				if err != nil {
					log.Errorf("failed to generate change: %v", err)
					return
				}

				if err := cli.SendChange(req); err != nil {
					log.Errorf("failed to send change: %v", err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}

	cli.RegisterToObject(nil, *objectID)

	wg.Wait()
}

// does some random changes against the existing JSON
// This is VERY specific to our existing JSON format obviously
func generateRandomChange(obj *client.Object) (*client.OutgoingChange, error) {

	var property string
	var data []byte
	var err error
	var a any
	rnd := rand.Intn(5)
	switch rnd {
	case 0:
		property = "features.0.attributes.field1"
		a = fmt.Sprintf("%d", rand.Intn(100000))
	case 1:
		property = "features.1.attributes.field2"
		a = fmt.Sprintf("%d", rand.Intn(100000))
	case 2:
		property = "fieldAliases.fieldName1"
		a = fmt.Sprintf("alias-%d", rand.Intn(100000))
	case 3:
		property = "spatialReference"
		a = fmt.Sprintf("sr-%d", rand.Intn(100000))
	case 4:
		property = "spatialReference"
		a = fmt.Sprintf("sr-%d", rand.Intn(100000))
	}

	data, err = json.Marshal(a)
	if err != nil {
		log.Errorf("Error marshalling JSON: %v", err)
		return nil, err
	}

	req := client.OutgoingChange{
		ObjectID:   obj.ObjectID,
		PropertyID: property,
		Data:       data,
	}

	return &req, nil
}
