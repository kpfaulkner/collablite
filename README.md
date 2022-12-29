# Collablite

Conflict free (mostly) data sharing service. Inspired by the Figma [post](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/)

aka, CRDT without the CRDT bit :)

## What is it?

Collablite is a service that allows you to share data between multiple clients. It is inspired by the Figma post on their multiplayer technology.
It is not a CRDT implementation, but it does use a similar approach to allow multiple clients to share data without conflict.

## How does it work?

There are a number of key features/conditions that this service provides:

- For a given object being edited (by multiple clients) the object exists ONLY in a single instance of the service. This may
  seem like a scaling issue in the future, but given that it's NOT expected that a LOT of changes will be happening to a single
  document at any one time, this should be safe. IF the instance of the service dies, then a new one can be fired up immediately
  and all clients can reconnect and continue. The state of the object at the time the service died is persisted so very little (if any)
  changes should be lost. Currently this is deemed acceptable.

  If the situation arises where a single instance of the service (for a specific object) is NOT sufficient and horizontal scaling would
  be required to meet the load, then a solution would be investigated then, but I don't want to go down that route yet.

- If more then one instance is required (to handle the general load, NOT specifically for one object) then the load balancer
  mechanism used will need to have some support for server affinity. If affinity cannot be handled then changes will NOT be
  shared correctly across clients.

- The resolution of concurrent conflicts of an object is that "last write wins". This is a simple approach but works well.
  Please see the Figma [post](https://www.figma.com/blog/how-figmas-multiplayer-technology-works/) for more details.


## Technologies used

TODO

## Architecture Diagram

TODO

## How to use it

An example client is provided in cmd/client/simpleproperty directory. This is a basic client that internally just treats
an object as a key/value pair.

An abbreviated version of this is:

```
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
)

// Simple key/value example...
func main() {
    host := "localhost:50051"
    objectID := "testobject"

    // new client to collablite server
    cli := client.NewClient(host)

    // create our keyvalue object that we're going to sync/manipulate
    kv := keyvalue.NewKeyValueObject(objectID)

    // register converters used to convert to/from KeyValueObject to the ClientObject
    // ConvertFromObject is to handle incoming changes. This is client specific. It will take the
    // ClientObject and convert it to the KeyValueObject.
    // ConvertToObject is for handling outgoing changes. It will take the KeyValueObject and convert it to
    // a ClientObject and will only send the changes (not the entire object) to the server.
    cli.RegisterConverters(kv.ConvertFromObject, kv.ConvertToObject)

    ctx := context.Background()
    // connect to server
    cli.Connect(ctx)

    // goroutine for listening for updates
    go cli.Listen(ctx)

    // register with the server for objectID we're interested in
    cli.RegisterToObject(nil, objectID)

    // client ID just to make sure we can track where each update is coming from. Purely for demo purposes.
    clientID := uuid.New().String()

    // wait group to make sure program doesn't exit before we're done
    wg := sync.WaitGroup{}
    wg.Add(1)

    // send updates to the server every 50 ms with random property changes
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

	wg.Wait()
}

```

