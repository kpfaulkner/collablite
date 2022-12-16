package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the client API for CollabLite service.
type Client struct {
	client proto.CollabLiteClient

	// is called when we receive a confirmation from the server.
	objectConfirmationCallback func(*ChangeConfirmation) error

	// stream to gRPC server
	stream proto.CollabLite_ProcessObjectChangesClient

	// used to track local changes (yet to be confirmed by server)
	unconfirmedLocalChanges map[string]int

	// and associated lock
	unconfirmedLock sync.Mutex

	// clientID used to help identify traffic from this client
	clientID string

	// number of conflicts, purely for stats collecting.
	numConflicts int
}

func NewClient(serverAddr string) *Client {

	// insecure for now.
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	client := proto.NewCollabLiteClient(conn)

	// Make channel size configurable FIXME(kpfaulkner)
	c := &Client{client: client,
		unconfirmedLocalChanges: make(map[string]int),
		clientID:                uuid.New().String(),
	}
	return c
}

// RegisterCallback is called whenever a change for the object being watched is received.
func (c *Client) RegisterCallback(cb func(*ChangeConfirmation) error) error {
	c.objectConfirmationCallback = cb
	return nil
}

// SendChange sends the change to the server for processing
func (c *Client) SendChange(outgoingChange *OutgoingChange) error {
	//t := time.Now()

	// convert to proto struct
	objChange := convertOutgoingChangeToProto(outgoingChange, c.clientID)
	// store change details for comparison with incoming confirmation
	objectProperty := fmt.Sprintf("%s-%s", objChange.ObjectId, objChange.PropertyId)
	c.unconfirmedLock.Lock()
	if _, ok := c.unconfirmedLocalChanges[objectProperty]; ok {
		//ids = append(ids, objChange.UniqueId)
		c.unconfirmedLocalChanges[objectProperty] = 1
	} else {
		c.unconfirmedLocalChanges[objectProperty] = c.unconfirmedLocalChanges[objectProperty] + 1
	}
	c.unconfirmedLock.Unlock()

	//tt := time.Now()
	if err := c.stream.Send(objChange); err != nil {
		// FIXME(kpfaulkner) shouldn't be fatal...
		log.Errorf("%v.Send(%v) = %v", c.stream, objChange, err)
		return err
	}
	//log.Debugf("stream send took %d ms", time.Since(tt).Milliseconds())

	//log.Debugf("send took: %v", time.Since(t).Milliseconds())

	return nil
}

// Connect creates the stream against the server
func (c *Client) Connect(ctx context.Context) error {

	stream, err := c.client.ProcessObjectChanges(ctx)
	if err != nil {
		log.Errorf("%v.ProcessObjectChanges(_) = _, %v", c.client, err)
		return err
	}

	c.stream = stream
	return nil
}

// GetObject returns the entire object from the server.
// Used for initial loading etc.
func (c *Client) GetObject(objectID string) ([]ChangeConfirmation, error) {
	resp, err := c.client.GetObject(context.Background(), &proto.GetRequest{
		ObjectId: objectID,
	})

	if err != nil {
		log.Errorf("failed to get object: %v", err)
		return nil, err
	}

	changes := make([]ChangeConfirmation, len(resp.Properties))
	i := 0
	for k, v := range resp.Properties {
		changes[i] = ChangeConfirmation{
			ObjectID:   resp.ObjectId,
			PropertyID: k,
			Data:       v,
		}
		i++
	}

	return changes, nil
}

// RegisterToObject sends a message to the server indicating that the client is listening for changes
// for a particular ObjectID. This only needs to be done once and only if we're passively listening and
// NOT sending data ourselves. If we're sending data we do not need to perform this.
func (c *Client) RegisterToObject(ctx context.Context, objectID string) error {

	// even if not sending changes... send an empty one to indicate what we want to listen to.
	req := OutgoingChange{
		ObjectID:   objectID,
		PropertyID: "",
		Data:       nil,
	}

	if err := c.SendChange(&req); err != nil {
		log.Errorf("failed to send change: %v", err)
		return err
	}

	return nil
}

// Listen will loop for incoming changes from the server. Any changes that are received
// and are NOT discarded (due to modifying the same object/property as a local change)
// will be passed to the callback registered via RegisterCallback
func (c *Client) Listen(ctx context.Context) error {

	//var origUniqueIDs []string
	var hasLocalChange bool

	count := 0
	// receive object confirmation
	for {
		objectConfirmation, err := c.stream.Recv()
		if err != nil {
			log.Errorf("%v.Recv() got error %v, want %v", c.stream, err, nil)
			return err
		}
		count++
		if count%100 == 0 {
			log.Debugf("Received %d", count)
		}
		objectProperty := fmt.Sprintf("%s-%s", objectConfirmation.ObjectId, objectConfirmation.PropertyId)

		// way too much happening in this lock. FIXME(kpfaulkner)
		c.unconfirmedLock.Lock()

		confirmedLocalChange := false

		// If this change doesn't match any property change performed locally, then allow it through and
		// call the callback
		if _, hasLocalChange = c.unconfirmedLocalChanges[objectProperty]; !hasLocalChange {

			//log.Debugf("not waiting on local confirmation %s", objectProperty)
			// do not have a local change for this object/property combo, so allow this through.
			c.objectConfirmationCallback(convertProtoToChangeConfirmation(objectConfirmation))
		} else {
			// check if change is from this client. If so, modify unconfirmedLocalChanges
			if objectConfirmation.UniqueId == c.clientID {
				if c.unconfirmedLocalChanges[objectProperty] > 0 {
					// does have local changes.. decrement count of changes.
					c.unconfirmedLocalChanges[objectProperty]--
				}

				// if no more left, then delete from map
				if c.unconfirmedLocalChanges[objectProperty] == 0 {
					delete(c.unconfirmedLocalChanges, objectProperty)
				}
				confirmedLocalChange = true
				c.objectConfirmationCallback(convertProtoToChangeConfirmation(objectConfirmation))
			}
		}

		/*
			confirmedLocalChange := false
			// If we've got here, then we know we have a local change for this object/property combo.
			for i, origUniqueID := range origUniqueIDs {
				// find the specific message.
				if origUniqueID == objectConfirmation.UniqueId {

					if objectProperty == "graphical-0-0" {
						log.Debugf("listen 0x0")
					}

					// remove from slice. Doing all this in a lock is stoooopid
					c.unconfirmedLocalChanges[objectProperty] = append(origUniqueIDs[:i], origUniqueIDs[i+1:]...)
					if len(c.unconfirmedLocalChanges[objectProperty]) == 0 {
						delete(c.unconfirmedLocalChanges, objectProperty)
					}

					confirmedLocalChange = true
					//log.Debugf("confirming local change %s", objectProperty)
					// this is our change... pass it through.
					//c.objectConfirmationCallback(convertProtoToChangeConfirmation(objectConfirmation))
					break
				}
			} */

		if hasLocalChange && !confirmedLocalChange {
			log.Debugf("CONFLICT.. but dropping %s", objectProperty)
			c.numConflicts++
		}
		// if we get here it means that we DO have a similar local change that has not been confirmed
		// so it means that we drop this. Our unconfirmed local change is still yet to arrive which
		// means it was generated after...  so this change will get wiped over anyway.
		c.unconfirmedLock.Unlock()
	}

	return nil
}

// GetConflictsCount returns number of conflicts the client has recorded
func (c *Client) GetConflictsCount() int {
	return c.numConflicts
}

// GetChangeCount lists the number of unconfirmed changes for client
func (c *Client) GetChangeCount() int {

	c.unconfirmedLock.Lock()
	count := 0
	for _, v := range c.unconfirmedLocalChanges {
		count += v // FIXME int change
	}
	c.unconfirmedLock.Unlock()
	return count
}

// convert models to proto structs
func convertOutgoingChangeToProto(outgoingChange *OutgoingChange, clientID string) *proto.ObjectChange {
	return &proto.ObjectChange{
		ObjectId:   outgoingChange.ObjectID,
		PropertyId: outgoingChange.PropertyID,
		Data:       outgoingChange.Data,

		// add unique id to track local changes.
		//UniqueId: fmt.Sprintf("%s-%s", clientID[:8], uuid.New().String()),
		UniqueId: clientID,
	}
}

func convertProtoToChangeConfirmation(confirmedChange *proto.ObjectConfirmation) *ChangeConfirmation {
	return &ChangeConfirmation{
		ObjectID:   confirmedChange.ObjectId,
		PropertyID: confirmedChange.PropertyId,
		Data:       confirmedChange.Data,
	}
}
