package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the client API for CollabLite service.
type Client struct {
	client proto.CollabLiteClient
}

func NewClient(serverAddr string) *Client {

	// insecure for now.
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	//defer conn.Close()

	client := proto.NewCollabLiteClient(conn)
	c := &Client{client: client}
	return c
}

func (c *Client) ProcessObjectChanges(objectChangeChan chan *proto.ObjectChange, objectConfirmationChan chan *proto.ObjectConfirmation) {
	//ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	ctx := context.Background()
	//defer cancel()

	stream, err := c.client.ProcessObjectChanges(ctx)
	if err != nil {
		log.Fatalf("%v.ProcessObjectChanges(_) = _, %v", c.client, err)
	}

	// first key is objectid-propertyid, value is our unique ID
	// So if there is a match for objectid-propertyid but it is NOT ours (unique ie) then drop
	// the incoming change. Given we might have multiple changes for the same obj/prop combo, the
	// value is a slice of unique ids... means that we'll need to manually trawl through the slice
	// FIXME(kpfaulkner) revisit this.
	unconfirmedLocalChanges := make(map[string][]string)
	unconfirmedLock := sync.Mutex{}
	
	// read object changes and send to server.
	go func() {
		var count int64 = 0
		t := time.Now()
		for objChange := range objectChangeChan {

			// store change details for comparison with incoming confirmation
			objectProperty := fmt.Sprintf("%s-%s", objChange.ObjectId, objChange.PropertyId)
			unconfirmedLock.Lock()
			if ids, ok := unconfirmedLocalChanges[objectProperty]; ok {
				ids = append(ids, objChange.UniqueId)
				unconfirmedLocalChanges[objectProperty] = ids
			} else {
				unconfirmedLocalChanges[objectProperty] = []string{objChange.UniqueId}
			}
			unconfirmedLock.Unlock()
			if err := stream.Send(objChange); err != nil {

				// FIXME(kpfaulkner) shouldn't be fatal...
				log.Fatalf("%v.Send(%v) = %v", stream, objChange, err)
			}
			//fmt.Printf("SENT %v\n", objChange)

			count++
			if count%100 == 0 {
				fmt.Printf("average send time: %v\n", time.Since(t).Milliseconds()/count)
			}

		}
	}()

	var origUniqueIDs []string
	var has bool

	// receive object confirmation
	for {
		objectConfirmation, err := stream.Recv()
		if err != nil {
			// FIXME(kpfaulkner) shouldn't be fatal...
			log.Fatalf("%v.Recv() got error %v, want %v", stream, err, nil)
		}

		if objectConfirmation == nil {
			fmt.Printf("dummy\n")
		}
		//fmt.Printf("RECV %v\n", objectConfirmation)

		objectProperty := fmt.Sprintf("%s-%s", objectConfirmation.ObjectId, objectConfirmation.PropertyId)

		// way too much happening in this lock. FIXME(kpfaulkner)
		unconfirmedLock.Lock()

		// see if we have local changes related to this.
		if origUniqueIDs, has = unconfirmedLocalChanges[objectProperty]; !has {
			// do not have a local change for this object/property combo, so allow this through.
			objectConfirmationChan <- objectConfirmation
		}

		// if this is our change, let it through.
		for i, origUniqueID := range origUniqueIDs {
			if origUniqueID == objectConfirmation.UniqueId {

				// remove from slice. Doing all this in a lock is stoooopid
				unconfirmedLocalChanges[objectProperty] = append(origUniqueIDs[:i], origUniqueIDs[i+1:]...)
				if len(unconfirmedLocalChanges[objectProperty]) == 0 {
					delete(unconfirmedLocalChanges, objectProperty)
				}
				objectConfirmationChan <- objectConfirmation
				break
			}
		}

		// if we get here it means that we DO have a similar local change that has not been confirmed
		// so it means that we drop this. Our unconfirmed local change is still yet to arrive which
		// means it was generated after...  so this change will get wiped over anyway.
		unconfirmedLock.Unlock()

	}

}
