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

	ctx := context.Background()

	// connect to server
	cli.Connect(ctx, *objectID)

	// goroutine for listening for updates
	go cli.Listen(ctx)

	// register with the server for objectID we're interested in
	cli.RegisterToObject(nil, *objectID)

	// client ID just to make sure we can track where each update is coming from. Purely for demo purposes.
	clientID := uuid.New().String()

	// wait group to make sure program doesn't exit before we're done
	wg := sync.WaitGroup{}
	wg.Add(1)

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
