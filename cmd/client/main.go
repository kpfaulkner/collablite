package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	client2 "github.com/kpfaulkner/collablite/client"
	"github.com/kpfaulkner/collablite/proto"
)

func main() {
	fmt.Printf("So it begins...\n")
	host := flag.String("host", "localhost:50051", "host:port of server")
	id := flag.String("id", "ken1", "id of client")
	send := flag.Bool("send", false, "send data to server")
	flag.Parse()

	localChangeChannel := make(chan *proto.ObjectChange, 1000)
	incomingChangesChannel := make(chan *proto.ObjectConfirmation, 1000)

	client := client2.NewClient(*host)

	wg := sync.WaitGroup{}
	wg.Add(1)

	clientID, _ := uuid.NewUUID()
	go client.ProcessObjectChanges(localChangeChannel, incomingChangesChannel)

	if *send {
		go func() {
			for i := 0; i < 1000000000; i++ {

				req := &proto.ObjectChange{
					ObjectId:   fmt.Sprintf("testobject1"),
					PropertyId: fmt.Sprintf("property-%03d", rand.Intn(100)),
					Data:       []byte(fmt.Sprintf("hello world-%s-%d", *id, i)),
					UniqueId:   clientID.String(),
				}

				localChangeChannel <- req
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// even if not sending changes... send an empty one to indicate what we want to listen to.
	req := &proto.ObjectChange{
		ObjectId:   fmt.Sprintf("testobject1"),
		PropertyId: "",
		Data:       nil,
		UniqueId:   "",
	}

	localChangeChannel <- req

	for resp := range incomingChangesChannel {
		fmt.Printf("confirmation: %v\n", resp)
	}

	wg.Wait()
}
