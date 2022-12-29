package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kpfaulkner/collablite/client"
	"github.com/kpfaulkner/collablite/client/converters/keyvalue"
	"github.com/kpfaulkner/collablite/cmd/common"
	log "github.com/sirupsen/logrus"
)

// Simple key/value example...
func main() {
	host := flag.String("host", "localhost:50051", "host:port of server")
	objectID := flag.String("objectid", "testobject1", "objectid of object to write/watch")
	send := flag.Bool("send", false, "send data to server")
	logLevel := flag.String("loglevel", "info", "Log Level: debug, info, warn, error")
	flag.Parse()

	common.SetLogLevel(*logLevel)

	cli := client.NewClient(*host)

	// create our keyvalue object that we're going to sync/manipulate
	kv := keyvalue.NewKeyValueObject(*objectID)

	// register converters used to convert to/from KeyValueObject to the ClientObject
	cli.RegisterConverters(kv.ConvertFromObject, kv.ConvertToObject)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx := context.Background()
	cli.Connect(ctx)
	go cli.Listen(ctx)

	cli.RegisterToObject(nil, *objectID)
	clientID := uuid.New().String()
	if *send {
		go func() {

			// do LOTS of changes :)
			for i := 0; i < 100000; i++ {

				kv.Lock.Lock()
				// do some random changes.
				kv.Properties[fmt.Sprintf("property-%03d", rand.Intn(100))] = []byte(fmt.Sprintf("hello world-%s-%d", clientID, i))
				kv.Lock.Unlock()
				if err := cli.SendObject(*objectID, kv); err != nil {
					log.Errorf("failed to send change: %v", err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}
